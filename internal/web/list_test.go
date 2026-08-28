package web_test

// The Issues view: the table under the toolbar. What is asserted here is what a
// reader can observe: how many issues the list says it holds, the order its
// rows stand in, what a column head claims about that order and where following
// it goes, where a row leads, and the words an empty list has. Never the class
// names that draw any of it.

import (
	"html"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/builtbystef/beaver-backlog/internal/core"
	"github.com/builtbystef/beaver-backlog/internal/issue"
)

func TestListSaysHowManyIssuesItIsShowing(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "First"})
	h := newHandler(t, dir)

	body := get(h, "/issues").Body.String()
	if !strings.Contains(body, "1 issue") || strings.Contains(body, "1 issues") {
		t.Errorf("a one-issue list does not say it is showing one:\n%s", body)
	}

	create(t, svc, core.Draft{Title: "Second"})
	create(t, svc, core.Draft{Title: "Third"})

	if body := get(h, "/issues").Body.String(); !strings.Contains(body, "3 issues") {
		t.Errorf("a three-issue list does not say it is showing three:\n%s", body)
	}
}

func TestFollowingAColumnHeaderSortsByItAndSaysSo(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	low := create(t, svc, core.Draft{Title: "Low", Priority: issue.PriorityLow})
	urgent := create(t, svc, core.Draft{Title: "Urgent", Priority: issue.PriorityUrgent})
	high := create(t, svc, core.Draft{Title: "High", Priority: issue.PriorityHigh})
	h := newHandler(t, dir)

	plain := get(h, "/issues").Body.String()
	if got := column(t, plain, "Priority").sort; got != "" {
		t.Fatalf("an unsorted list already claims to be sorted by priority: %q", got)
	}

	sorted := get(h, column(t, plain, "Priority").href).Body.String()
	if got, want := listRows(t, sorted), []string{urgent.ID, high.ID, low.ID}; !slices.Equal(got, want) {
		t.Errorf("following the Priority header ordered %v, want %v", got, want)
	}
	if got := column(t, sorted, "Priority").sort; got != "ascending" {
		t.Errorf("the sorted list says %q about its Priority column, want %q", got, "ascending")
	}

	reversed := get(h, column(t, sorted, "Priority").href).Body.String()
	if got, want := listRows(t, reversed), []string{low.ID, high.ID, urgent.ID}; !slices.Equal(got, want) {
		t.Errorf("following the Priority header again ordered %v, want %v", got, want)
	}
	if got := column(t, reversed, "Priority").sort; got != "descending" {
		t.Errorf("the reversed list says %q about its Priority column, want %q", got, "descending")
	}
}

// A sorted list is a place, not a session: the whole of what the reader is
// looking at rides in the address, filters included, so the address alone
// reproduces the view.
func TestASortedListIsLinkableFiltersAndAll(t *testing.T) {
	dir := newStore(t)
	svc := open(t, dir)
	create(t, svc, core.Draft{Title: "Zebra", Labels: []string{"ui"}})
	create(t, svc, core.Draft{Title: "Apple", Labels: []string{"ui"}})
	create(t, svc, core.Draft{Title: "Mango"})
	h := newHandler(t, dir)

	to := column(t, get(h, "/issues?label=ui").Body.String(), "Title").href
	if !strings.Contains(to, "label=ui") {
		t.Fatalf("sorting a filtered list drops the filter: %s", to)
	}

	followed := get(h, to).Body.String()
	rows := listRows(t, followed)
	if len(rows) != 2 {
		t.Fatalf("the filtered list holds %d rows, want the 2 labelled issues", len(rows))
	}
	if got := titles(t, followed); !slices.Equal(got, []string{"Apple", "Zebra"}) {
		t.Errorf("sorting by title ordered %v, want Apple before Zebra", got)
	}

	// The same address typed rather than followed is the same view.
	pasted := get(h, "/issues?label=ui&sort=title").Body.String()
	if got := listRows(t, pasted); !slices.Equal(got, rows) {
		t.Errorf("the pasted address lists %v; following the header listed %v", got, rows)
	}
	if got := column(t, pasted, "Title").sort; got != "ascending" {
		t.Errorf("the pasted address says %q about its Title column, want %q", got, "ascending")
	}
}

// A row is a doorway to the issue it names, the way a card is, and the links
// inside it keep their own meaning, so ui.js has something to tell apart.
func TestARowNamesTheIssueItLeadsTo(t *testing.T) {
	dir := newStore(t)
	target := create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})

	row := rowFor(t, get(newHandler(t, dir), "/issues").Body.String(), target.ID)

	if !strings.Contains(row, `data-href="/issues/`+target.ID+`"`) {
		t.Errorf("the row for %s does not say where it leads:\n%s", target.ID, row)
	}
	var to []string
	for _, a := range anchor.FindAllStringSubmatch(row, -1) {
		to = append(to, read(a).href)
	}
	if !slices.Contains(to, "/issues/"+target.ID) {
		t.Errorf("the row for %s holds no link of its own to the issue: %v", target.ID, to)
	}
}

func TestAFilterMatchingNothingSaysSoRatherThanLookingEmpty(t *testing.T) {
	dir := newStore(t)
	create(t, open(t, dir), core.Draft{Title: "Fix flag parsing"})

	body := get(newHandler(t, dir), "/issues?label=nothing-wears-this").Body.String()

	if !strings.Contains(body, "No issue matches these filters") {
		t.Errorf("a filter matching nothing does not say so:\n%s", body)
	}
	if strings.Contains(body, "No issues yet") {
		t.Error("a filter matching nothing claims the store is empty")
	}
}

func TestAnEmptyStoreOffersTheWayToCreateAnIssue(t *testing.T) {
	body := get(newHandler(t, newStore(t)), "/issues").Body.String()

	if !strings.Contains(body, "No issues yet") {
		t.Errorf("an empty store does not say it is empty:\n%s", body)
	}
	if !strings.Contains(body, `href="/issues/new"`) {
		t.Errorf("an empty list offers no way to create an issue:\n%s", body)
	}
}

var (
	listTable  = regexp.MustCompile(`(?s)<table[^>]*>(.*?)</table>`)
	headerCell = regexp.MustCompile(`(?s)<th([^>]*)>(.*?)</th>`)
	tableBody  = regexp.MustCompile(`(?s)<tbody[^>]*>(.*?)</tbody>`)
	tableRow   = regexp.MustCompile(`(?s)<tr[^>]*data-issue="([a-z0-9]+)"[^>]*>(.*?)</tr>`)
	sortState  = regexp.MustCompile(`aria-sort="([a-z]+)"`)
)

// heading is one column head read back off the table: what it says, where
// following it goes, and what it claims about the order the rows stand in.
type heading struct {
	label string
	href  string
	sort  string
}

func column(t *testing.T, body, label string) heading {
	t.Helper()
	for _, cell := range headerCell.FindAllStringSubmatch(tableOf(t, body), -1) {
		attrs, inner := cell[1], cell[2]
		words := strings.Fields(html.UnescapeString(stripTags.ReplaceAllString(inner, " ")))
		if len(words) == 0 || words[0] != label {
			continue
		}
		h := heading{label: label}
		if m := sortState.FindStringSubmatch(attrs); m != nil {
			h.sort = m[1]
		}
		if a := anchor.FindStringSubmatch(inner); a != nil {
			h.href = html.UnescapeString(a[1])
		}
		return h
	}
	t.Fatalf("no %q column in the table:\n%s", label, body)
	return heading{}
}

// listRows are the issue IDs the table's rows name, in the order they stand.
func listRows(t *testing.T, body string) []string {
	t.Helper()
	var out []string
	for _, r := range rowsOf(t, body) {
		out = append(out, r[1])
	}
	return out
}

// titles are the issue titles the rows carry, in the order they stand: what
// an ordering by title is read back off.
func titles(t *testing.T, body string) []string {
	t.Helper()
	var out []string
	for _, r := range rowsOf(t, body) {
		for _, a := range anchor.FindAllStringSubmatch(r[2], -1) {
			if one := read(a); one.label != "" && one.label != r[1] {
				out = append(out, one.label)
				break
			}
		}
	}
	return out
}

func rowFor(t *testing.T, body, id string) string {
	t.Helper()
	for _, r := range rowsOf(t, body) {
		if r[1] == id {
			return r[0]
		}
	}
	t.Fatalf("no row for %s in the table:\n%s", id, body)
	return ""
}

func tableOf(t *testing.T, body string) string {
	t.Helper()
	m := listTable.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no issues table on the page:\n%s", body)
	}
	return m[1]
}

func rowsOf(t *testing.T, body string) [][]string {
	t.Helper()
	m := tableBody.FindStringSubmatch(tableOf(t, body))
	if m == nil {
		t.Fatalf("the issues table has no body:\n%s", body)
	}
	rows := tableRow.FindAllStringSubmatch(m[1], -1)
	if len(rows) == 0 {
		t.Fatalf("the issues table holds no rows:\n%s", m[1])
	}
	return rows
}
