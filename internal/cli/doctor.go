package cli

// cmdDoctor asks the core for the store's health, renders the report, and exits
// non-zero while any problem remains. Which files are wrong and which wrongs are
// mechanically safe to repair is the core's business; --fix only passes the
// request along.
func cmdDoctor(env Env, args []string) int {
	fs, formatFlag := newFlagSet(env, "doctor")
	fixFlag := fs.Bool("fix", false, "repair lint-class problems (filename drift); never removes data")
	pos, ok := parseArgs(fs, args)
	if !ok {
		return exitUsage
	}
	if len(pos) > 0 {
		errf(env, "doctor takes no arguments")
		return exitUsage
	}
	format, err := resolveFormat(env, *formatFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitUsage
	}

	svc, err := open(env)
	if err != nil {
		return coreError(env, err)
	}
	// No warnings to report: doctor's scan hands back unusable files as findings,
	// so nothing is skipped silently and nothing is named twice.
	rep, err := svc.Doctor(*fixFlag)
	if err != nil {
		errf(env, "%v", err)
		return exitError
	}
	if err := renderReport(env, rep, format, *fixFlag); err != nil {
		errf(env, "%v", err)
		return exitError
	}
	// Non-zero while problems remain, so scripts can branch on store health
	// without parsing the report.
	if rep.Problems() > 0 {
		return exitError
	}
	return exitOK
}
