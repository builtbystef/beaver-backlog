package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/builtbystef/beaver-backlog/internal/web"
)

// defaultPort is where the web UI listens unless --port says otherwise: a fixed
// default so the URL is memorable across sessions.
const defaultPort = 2328

// portScanLimit bounds how many consecutive ports the default listen tries
// before giving up. Sequential ports keep URLs memorable when several projects
// serve at once; the bound keeps a pathological machine from a long stall.
const portScanLimit = 10

// shutdownGrace is how long an interrupted server waits for in-flight requests
// before dropping them.
const shutdownGrace = 5 * time.Second

// cmdServe runs the local web UI in the foreground until the process is
// interrupted. It resolves the actor once, here at launch, so every write the
// browser makes is attributed to whoever started the server; the browser has no
// identity of its own. The socket is loopback-only and there is no option to
// widen it: the UI is one person's view of their own files, with no auth to put
// in front of it.
func cmdServe(env Env, args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	portFlag := fs.Int("port", defaultPort, "port to listen on (0 picks a free one)")
	asFlag := fs.String("as", "", "attribute web writes to this actor (overrides identity detection)")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	portChosen := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portChosen = true
		}
	})
	if len(pos) > 0 {
		errf(env, "serve takes no arguments (did you mean --port %s?)", pos[0])
		return exitUsage
	}
	if *portFlag < 0 || *portFlag > 65535 {
		errf(env, "invalid port %d (want 0-65535)", *portFlag)
		return exitUsage
	}

	// The store is checked first: no store means no server, before an identity
	// prompt or a socket.
	if _, err := open(env); err != nil {
		return coreError(env, err)
	}
	me, err := resolveActor(env, *asFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	handler, err := web.New(web.Config{WorkDir: env.WorkDir, Actor: me.name, CoreOptions: env.CoreOptions})
	if err != nil {
		return coreError(env, err)
	}

	ln, err := listenLoopback(env, *portFlag, portChosen)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	fmt.Fprintf(env.Stdout, "beaver: serving http://%s (press Ctrl-C to stop)\n", ln.Addr())
	return serveUntilInterrupt(env, ln, handler)
}

// listenLoopback binds the loopback socket for the web UI. A port the user
// chose is honored or fails outright: they asked for that port and no other.
// The default port instead scans forward to the next free one, so serves for
// several projects coexist without anyone picking ports; the note tells the
// user why the URL is not the usual one.
func listenLoopback(env Env, port int, chosen bool) (net.Listener, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err == nil || chosen {
		return ln, err
	}
	for p := port + 1; p < port+portScanLimit; p++ {
		next, retryErr := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p)))
		if retryErr == nil {
			fmt.Fprintf(env.Stdout, "beaver: port %d is taken (another beaver serve?); using %d\n", port, p)
			return next, nil
		}
	}
	// The original error names the port the user expected, which is the one
	// worth explaining when the whole scan comes up empty.
	return nil, err
}

// serveUntilInterrupt runs the server until the environment's context is
// cancelled, which is the interrupt the binary translates, then lets in-flight
// requests finish before returning.
func serveUntilInterrupt(env Env, ln net.Listener, handler http.Handler) int {
	ctx := env.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// ReadHeaderTimeout only guards against a stuck local client holding a
	// connection open; there is no untrusted traffic on a loopback socket.
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	stopped := make(chan error, 1)
	go func() { stopped <- srv.Serve(ln) }()

	select {
	case err := <-stopped:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errf(env, "%v", err)
			return exitError
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), shutdownGrace)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			errf(env, "%v", err)
			return exitError
		}
	}
	return exitOK
}
