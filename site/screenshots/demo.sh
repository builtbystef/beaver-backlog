#!/usr/bin/env bash
# Builds the Lantern demo store the screenshots are taken against: thirty
# issues across every state, with priorities, labels, assignees, parents,
# dependencies, and attributed notes, on a timeline spread over six weeks.
#
# usage: BEAVER=./beaver site/screenshots/demo.sh /tmp/lantern
#        (cd /tmp/lantern && beaver serve --port 2340 --as stefan)
set -euo pipefail
S=$(cd "$(dirname "$0")" && pwd)
B=${BEAVER:-beaver}
D=${1:?usage: demo.sh <dir>}
[ -e "$D" ] && { echo "demo.sh: $D exists; remove it first" >&2; exit 2; }
mkdir -p "$D"; cd "$D"
git init -q .
export BEAVER_BACKLOG_ACTOR=stefan
$B init >/dev/null
printf 'name: Lantern\n' >> .beaver/config.yml

id() { $B create "$@" --format json | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])'; }

# ---- Epics ------------------------------------------------------------
READER=$(id "Reader mode for saved articles" --priority high --label feature --body "$(cat <<'MD'
Saved articles open in a clean reading view instead of the original page.

## Why

Readers save an article to come back to it later. When they do, they get the
publisher's page again: cookie banners, autoplay video, three fonts. The whole
point of saving something was to read it in peace.

## Scope

- Extract the article text and images once, when the article is saved
- Render it in a reading layout with typography the reader can adjust
- Keep the original link one click away

## Out of scope

Comments, paywalled content, and anything that needs the publisher's scripts.
MD
)")
IMPORT=$(id "Import from Pocket and Instapaper" --priority medium --label feature --body "$(cat <<'MD'
Most people arrive with a reading list somewhere else. Bring it along.

Both services export a flat file (Pocket a CSV, Instapaper a CSV or HTML). One
importer, two parsers, and a page that shows the import as it runs.

- [x] Look at real exports from both services
- [ ] Pocket CSV
- [ ] Instapaper CSV and HTML
- [ ] Progress page
MD
)")
OFFLINE=$(id "Offline support" --priority low --label feature --body "$(cat <<'MD'
The reading list should be usable on a train. Article text is small and already
extracted, so caching it is cheap; the hard part is what happens to read state
and new saves made while offline.
MD
)")

# ---- Done -------------------------------------------------------------
CI=$(id "Set up CI with lint and race tests" --priority medium --label infra --body "GitHub Actions: \`go vet\`, \`golangci-lint\`, and \`go test -race ./...\` on every push. Cache the module download between runs.")
OPML=$(id "Parse OPML feed exports" --priority medium --label feature --body "$(cat <<'MD'
Every feed reader exports OPML. Read the nested outline, keep the folder names
as tags, and skip entries without an \`xmlUrl\`.

\`\`\`xml
<outline text="Go" title="Go">
  <outline type="rss" text="The Go Blog" xmlUrl="https://go.dev/blog/feed.atom"/>
</outline>
\`\`\`
MD
)")
SCHED=$(id "Fetch feeds on a schedule" --priority high --label feature --body "A background fetcher polls every feed on an interval, honouring \`ETag\` and \`Last-Modified\` so unchanged feeds cost one conditional request." --depends-on "$OPML")
STRIP=$(id "Strip tracking parameters from saved URLs" --priority medium --label bug,privacy --body "$(cat <<'MD'
Saved links keep \`utm_*\`, \`fbclid\`, \`gclid\` and friends. Two saves of the
same article from different newsletters end up as two entries.

Strip the known tracking parameters at save time, before the URL is stored or
compared.
MD
)")
CARD=$(id "Design the article card" --priority medium --label ux --body "Title, source, saved-at, and an optional lead image. One line of excerpt. Must read well at 320px wide.")
QUICK=$(id "Write the quick start guide" --priority low --label docs --body "Install, add a feed, save an article, read it. Under five minutes, with screenshots.")

# ---- Cancelled --------------------------------------------------------
SAFARI=$(id "Browser extension for Safari" --priority low --label feature --body "A save-to-Lantern button for Safari. Needs an Apple developer account and a signed build for every release.")
PG=$(id "Migrate from SQLite to Postgres" --priority low --label infra --body "Considered for multi-user deployments. SQLite in WAL mode handles the load we actually see.")

# ---- In progress and todo ---------------------------------------------
EXTRACT=$(id "Extract article text with readability heuristics" --priority high --label feature --parent "$READER" --body "$(cat <<'MD'
Port the readability scoring approach: score every block element by text
density, link density, and class name hints, then keep the best-scoring
subtree and its siblings above a threshold.

## Plan

- [x] Fetch and parse the page with the same client the feed fetcher uses
- [x] Score candidate nodes
- [ ] Clean the winning subtree (drop forms, share buttons, empty paragraphs)
- [ ] Rewrite relative image URLs to absolute ones
- [ ] Golden tests against twenty real articles

## Test corpus

Keep the HTML fixtures under \`internal/extract/testdata\`. No article over
200KB; trim them by hand if needed.
MD
)")
TYPO=$(id "Typography settings for reader mode" --priority medium --label ux --parent "$READER" --depends-on "$EXTRACT" --body "Font family (serif, sans, mono), size in three steps, and line width. Stored per browser, applied before the first paint so nothing jumps.")
DARK=$(id "Dark palette for the reader" --priority medium --label ux --parent "$READER" --depends-on "$TYPO" --body "Follow the system preference, with a manual override next to the typography settings. Images get a slight dim so a white photo does not glare.")
RTIME=$(id "Estimated reading time on article cards" --priority low --label feature --depends-on "$EXTRACT" --body "Word count of the extracted text divided by 230 words a minute, rounded up. Show it on the card and at the top of the reader.")
SEARCH=$(id "Full-text search over saved articles" --priority high --label feature --depends-on "$EXTRACT" --body "$(cat <<'MD'
Search the extracted text, not just titles. SQLite FTS5 with the porter
tokenizer is enough for a personal list.

Ranking: title match first, then body, most recently saved first among equals.
Highlight the matching snippet on the result card.
MD
)")
DEDUPE=$(id "Dedupe articles saved from AMP and canonical URLs" --priority high --label bug --depends-on "$STRIP" --body "$(cat <<'MD'
Saving \`example.com/amp/story\` and \`example.com/story\` yields two entries.
Resolve the \`<link rel=canonical>\` at save time and compare on that.

Steps to reproduce:

1. Save an article from a Google AMP viewer link
2. Save the same article from the publisher's site
3. Two cards
MD
)")
POCKET=$(id "Import Pocket CSV exports" --priority medium --label feature --parent "$IMPORT" --body "$(cat <<'MD'
Pocket's export is a CSV with \`title,url,time_added,tags,status\`. Times are
Unix seconds; tags are pipe-separated; status is \`unread\` or \`archive\`.

Map \`archive\` to read, keep the tags, and skip rows whose URL fails to parse
with a count in the summary.
MD
)")
INSTA=$(id "Import Instapaper exports" --priority medium --label feature --parent "$IMPORT" --depends-on "$POCKET" --body "Instapaper exports CSV (\`URL,Title,Selection,Folder,Timestamp\`) or an HTML bookmarks file. Reuse the row pipeline from the Pocket importer; folders become tags.")
PROGRESS=$(id "Import progress page" --priority low --label ux --parent "$IMPORT" --depends-on "$POCKET,$INSTA" --body "A page that shows rows imported, skipped, and failed as the import runs, then a summary with a link to each failed row's reason.")
SW=$(id "Service worker caches extracted article text" --priority low --label feature --parent "$OFFLINE" --depends-on "$EXTRACT" --body "Cache the reader view for the fifty most recently saved articles. Images are cached only when they are under 200KB.")
SYNC=$(id "Sync read state when back online" --priority low --label feature --parent "$OFFLINE" --depends-on "$SW" --body "Queue read/unread toggles made offline and replay them in order once a request succeeds. Last write wins; no merging.")
KEYS=$(id "Keyboard navigation in the reading list" --priority medium --label ux --body "$(cat <<'MD'
\`j\`/\`k\` to move, \`o\` to open, \`e\` to archive, \`/\` to focus search. Show
the bindings behind \`?\`.

Focus ring must be visible in both palettes.
MD
)")
RATE=$(id "Feed fetch retries hammer a host that returns 429" --priority urgent --label bug --depends-on "$SCHED" --body "$(cat <<'MD'
A host that answers \`429 Too Many Requests\` is retried immediately, three
times, on every fetch cycle. One user reports being blocked by a publisher.

## Expected

Honour \`Retry-After\` when present; otherwise back off exponentially per host,
capped at six hours.

## Log excerpt

\`\`\`
fetch feed=1842 status=429 attempt=1
fetch feed=1842 status=429 attempt=2
fetch feed=1842 status=429 attempt=3
\`\`\`
MD
)")
LOGIN=$(id "Rate limit login attempts" --priority high --label security --body "Five failures per username per fifteen minutes, then a fixed delay. Log the lockout without the password.")
SCROLL=$(id "Feed list scrolls to top after marking an item read" --priority medium --label bug,ux --body "Marking an item read re-renders the list and loses the scroll position. Keep the position, or better, update the one row in place.")
EXPORT=$(id "Export the reading list as JSON" --priority low --label feature --body "Everything the importers accept, in one file, so leaving is as easy as arriving.")
SCHEDDOC=$(id "Document the fetch schedule settings" --priority low --label docs --depends-on "$SCHED" --body "Interval, per-host concurrency, and the conditional request behaviour, with the defaults stated.")
LAZY=$(id "Lazy-load images in the reading list" --priority medium --label perf --body "Cards below the fold load their lead image on scroll. Use native \`loading=lazy\` with explicit dimensions so the layout does not shift.")
TAGS=$(id "Tag saved articles" --priority medium --label feature --body "Free-form tags on a saved article, a tag filter in the list, and the same tags carried through import and export.")

# ---- Lifecycle --------------------------------------------------------
for i in $CI $CARD; do $B start "$i" --as mira >/dev/null; $B done "$i" >/dev/null; done
for i in $OPML $QUICK; do $B start "$i" --as stefan >/dev/null; $B done "$i" >/dev/null; done
$B start "$SCHED" --as ola >/dev/null; $B done "$SCHED" >/dev/null
$B start "$STRIP" --as claude >/dev/null; $B done "$STRIP" >/dev/null
$B cancel "$SAFARI" >/dev/null; $B cancel "$PG" >/dev/null

$B start "$READER" --as stefan >/dev/null
$B start "$EXTRACT" --as ola >/dev/null
$B start "$KEYS" --as mira >/dev/null
$B start "$RATE" --as stefan >/dev/null
$B start "$POCKET" --as claude >/dev/null
$B update "$SEARCH" --assignee ola >/dev/null
$B update "$LOGIN" --assignee mira >/dev/null
$B update "$IMPORT" --assignee claude >/dev/null

# ---- Notes ------------------------------------------------------------
$B note "$EXTRACT" --as ola "Scoring works on the first ten fixtures. Recipe sites are the hard case: the ingredient list scores low on text density and gets dropped. Trying a bonus for lists directly under the best candidate." >/dev/null
$B note "$EXTRACT" --as stefan "Fine to ship without recipe sites if the golden tests are green on the news and blog fixtures. Track recipes as a follow-up." >/dev/null
$B note "$EXTRACT" --as ola "Agreed. Cleaning pass is next; relative image URLs after that." >/dev/null
$B note "$RATE" --as stefan "Reproduced against a local server that always answers 429. The retry loop ignores Retry-After entirely; the header is parsed but never read." >/dev/null
$B note "$RATE" --as mira "Publisher unblocked us after an email. Let's get this out in a patch release before the next fetch cycle finds another one." >/dev/null
$B note "$POCKET" --as claude "Claimed. Real export from stefan has 1,412 rows, 9 with URLs that fail to parse (all mailto: links). Skipping those with a count as the spec says." >/dev/null
$B note "$POCKET" --as stefan "Looks right. Make sure the time_added column survives as the saved-at date; the list sorts on it." >/dev/null
$B note "$STRIP" --as claude "Done. The list of parameters lives in internal/save/tracking.go; adding one is a one-line change with a test." >/dev/null
$B note "$KEYS" --as mira "Bindings are in. The focus ring in the dark palette is too faint against cards; borrowing the accent colour from the reader settings." >/dev/null
$B note "$SEARCH" --as ola "Prototype with FTS5 indexes 2,000 articles in under a second. Snippet highlighting is the remaining piece." >/dev/null
$B note "$DEDUPE" --as stefan "Blocked on nothing now that tracking parameters are stripped. Ready for whoever picks it up." >/dev/null
$B note "$PG" --as stefan "Cancelled: SQLite in WAL mode handles every deployment we know of. Revisit if a multi-user request shows up." >/dev/null
$B note "$SAFARI" --as mira "Cancelled for now; the bookmarklet covers Safari well enough and costs nothing to ship." >/dev/null
$B note "$READER" --as stefan "Extraction is the long pole. Typography and the dark palette are small once the text is there." >/dev/null

sed -i 's/\\`/`/g' .beaver/issues/*.md

# ---- Spread the timeline over the last six weeks ------------------------
python3 - <<'PY'
import os, re, glob, random, datetime as dt
random.seed(7)
now = dt.datetime(2026, 9, 2, 15, 40, tzinfo=dt.timezone.utc)
files = sorted(glob.glob('.beaver/issues/*.md'), key=os.path.getmtime)
n = len(files)
for i, path in enumerate(files):
    src = open(path).read()
    created = now - dt.timedelta(days=42 - i * (36 / n), hours=random.randint(0, 9), minutes=random.randint(0, 59))
    updated = created + dt.timedelta(days=random.randint(1, 10), hours=random.randint(0, 23))
    if updated > now: updated = now - dt.timedelta(hours=random.randint(1, 30))
    fmt = lambda d: d.strftime('%Y-%m-%dT%H:%M:%SZ')
    src = re.sub(r'^created: .*$', f'created: {fmt(created)}', src, count=1, flags=re.M)
    src = re.sub(r'^updated: .*$', f'updated: {fmt(updated)}', src, count=1, flags=re.M)
    # notes: spread between created and updated, in order
    stamps = re.findall(r'^\*\*\w+\*\* — (\S+)$', src, flags=re.M)
    if stamps:
        span = (updated - created)
        for k, old in enumerate(stamps):
            t = created + span * ((k + 1) / (len(stamps) + 1))
            src = src.replace(old, fmt(t), 1)
    open(path, 'w').write(src)
PY
$B doctor --format human
