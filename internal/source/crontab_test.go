package source

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/hzerrad/cronkit/internal/cronx"
	"github.com/hzerrad/cronkit/internal/inventory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func extractCrontab(t *testing.T, name, content string) []inventory.Item {
	t.Helper()
	src := NewCrontabSource()

	fsys := fstest.MapFS{name: &fstest.MapFile{Data: []byte(content)}}
	info, err := fs.Stat(fsys, name)
	require.NoError(t, err)

	unit := Unit{Path: name, Info: info}
	require.True(t, src.Match(unit), "the fixture should match the crontab source")

	items, err := src.Extract(unit, fsys)
	require.NoError(t, err)
	return items
}

func TestCrontabSource_ExtractsJobsWithProvenance(t *testing.T) {
	items := extractCrontab(t, "ops/backup.cron", `# nightly work
0 2 * * * /usr/bin/backup.sh  # takes about an hour

MAILTO=ops@example.com
*/15 * * * * /usr/bin/poll.sh
`)

	require.Len(t, items, 2)

	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "crontab", items[0].SourceID)
	assert.Equal(t, "vixie", items[0].Dialect)
	assert.Equal(t, "/usr/bin/backup.sh", items[0].Command)
	assert.True(t, items[0].Shell, "a crontab command is a real shell command")
	assert.Equal(t, "takes about an hour", items[0].Comment)
	assert.Equal(t, inventory.StateActive, items[0].State)
	assert.Equal(t, "ops/backup.cron", items[0].Locator.File)
	assert.Equal(t, 2, items[0].Locator.Line)
	assert.Empty(t, items[0].Locator.Path, "a crontab has no structural path")

	assert.Equal(t, "*/15 * * * *", items[1].Expression)
	assert.Equal(t, 5, items[1].Locator.Line)
}

func TestCrontabSource_InvalidExpressionIsRecordedNotDropped(t *testing.T) {
	items := extractCrontab(t, "a.cron", "99 99 * * * /usr/bin/broken.sh\n")

	require.Len(t, items, 1, "an unparseable job is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

// TestCrontabSource_UnrecognisedFieldValueIsRecordedNotDropped covers an unrecognised field value.
func TestCrontabSource_UnrecognisedFieldValueIsRecordedNotDropped(t *testing.T) {
	items := extractCrontab(t, "a.cron", "0 9 * * lun /usr/bin/fr\n")

	require.Len(t, items, 1, "an expression with one unrecognised field is still inventoried, not silently dropped")
	assert.Equal(t, "0 9 * * lun", items[0].Expression)
	assert.Equal(t, "/usr/bin/fr", items[0].Command)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestCrontabSource_MalformedEveryDurationIsRecordedNotDropped(t *testing.T) {
	items := extractCrontab(t, "a.cron", "@every nonsense /d.sh\n")

	require.Len(t, items, 1, "a malformed @every is still inventoried, not silently dropped")
	assert.Equal(t, "@every nonsense", items[0].Expression)
	assert.Equal(t, "/d.sh", items[0].Command)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestCrontabSource_MalformedAliasIsRecordedNotDropped(t *testing.T) {
	// A leading "@" is still an unambiguous schedule attempt and must be reported as StateInvalid.
	items := extractCrontab(t, "a.cron", "@dialy /f.sh\n")

	require.Len(t, items, 1, "a malformed descriptor is still inventoried, not silently dropped")
	assert.Equal(t, "@dialy", items[0].Expression)
	assert.Equal(t, "/f.sh", items[0].Command)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestCrontabSource_CronDMalformedAliasSplitsTheUserField(t *testing.T) {
	// A malformed descriptor in a cron.d fragment must still split its USER field like a well-formed one.
	items := extractCrontab(t, "etc/cron.d/backup", "@dialy root /f.sh\n")

	require.Len(t, items, 1, "a malformed descriptor is still inventoried, not silently dropped")
	assert.Equal(t, "@dialy", items[0].Expression)
	assert.Equal(t, "root", items[0].RunAs, "the user field is split out, matching the well-formed job path")
	assert.Equal(t, "/f.sh", items[0].Command)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason, "the reason still names the malformed descriptor, not the user split")
}

func TestCrontabSource_CronDMalformedAliasMissingUserFieldKeepsCommandWhole(t *testing.T) {
	// When the user-field split fails, the descriptor's own Reason must win and the command stays intact.
	items := extractCrontab(t, "etc/cron.d/backup", "@dialy /f.sh\n")

	require.Len(t, items, 1)
	assert.Empty(t, items[0].RunAs)
	assert.Equal(t, "/f.sh", items[0].Command)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestCrontabSource_DescriptorsWithNoCommandAreAllReportedTheSameWay(t *testing.T) {
	// A descriptor with nothing after it must come back as exactly one StateInvalid item.
	for _, line := range []string{"@every 1h\n", "@reboot\n", "@daily\n"} {
		t.Run(line, func(t *testing.T) {
			items := extractCrontab(t, "a.cron", line)

			require.Len(t, items, 1, "a descriptor with no command is still inventoried, not dropped")
			assert.Equal(t, inventory.StateInvalid, items[0].State)
			assert.Empty(t, items[0].Command)
			assert.Contains(t, items[0].Reason, "no command")
		})
	}
}

func TestCrontabSource_CronDStillIgnoresProseStartingWithoutAt(t *testing.T) {
	// Prose sharing a cron.d directory with real fragments, but not opening with "@", must still yield nothing.
	items := extractCrontab(t, "etc/cron.d/README", `See the on-call runbook before touching any file in this directory.
`)

	assert.Empty(t, items, "prose with no leading @ must still be skipped, not reported as a malformed descriptor")
}

func TestCrontabSource_CronDIgnoresProseStartingWithAtButStillReportsADescriptorTypo(t *testing.T) {
	// A bare "@" used the way "at" is in ordinary English must not be misread as an attempted descriptor.
	items := extractCrontab(t, "etc/cron.d/README", `@ the team in slack if you need help with any of this stuff right now
`)

	assert.Empty(t, items, "a bare leading @ used as prose must not be reported as a malformed descriptor")
}

func TestCrontabSource_CronDStillReportsAMultiWordDescriptorTypo(t *testing.T) {
	// A first-field descriptor typo is still unambiguous, even amid several words of trailing prose.
	items := extractCrontab(t, "etc/cron.d/README", `@dialy please tell the team if this ever fires
`)

	require.Len(t, items, 1, "a multi-word line opening with a descriptor typo is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
}

func TestCrontabSource_AppliesCronTZSetBeforeJobs(t *testing.T) {
	items := extractCrontab(t, "a.cron", `CRON_TZ=Europe/Paris
0 2 * * * /usr/bin/backup.sh
*/15 * * * * /usr/bin/poll.sh
`)

	require.Len(t, items, 2)
	assert.Equal(t, "Europe/Paris", items[0].Timezone)
	assert.Equal(t, "Europe/Paris", items[1].Timezone)
}

func TestCrontabSource_AppliesTZWhenNoCronTZIsSet(t *testing.T) {
	items := extractCrontab(t, "a.cron", `TZ=America/New_York
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 1)
	assert.Equal(t, "America/New_York", items[0].Timezone)
}

func TestCrontabSource_CronTZTakesPrecedenceOverTZ(t *testing.T) {
	items := extractCrontab(t, "a.cron", `TZ=America/New_York
CRON_TZ=Europe/Paris
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 1)
	assert.Equal(t, "Europe/Paris", items[0].Timezone,
		"CRON_TZ takes precedence over TZ when both are set")
}

func TestCrontabSource_ZoneChangePartwayDownKeepsEarlierJobsAtTheOldZone(t *testing.T) {
	items := extractCrontab(t, "a.cron", `CRON_TZ=Europe/Paris
0 2 * * * /usr/bin/before.sh
CRON_TZ=Asia/Tokyo
0 3 * * * /usr/bin/after.sh
`)

	require.Len(t, items, 2)
	assert.Equal(t, "Europe/Paris", items[0].Timezone,
		"a job scheduled before the reassignment keeps the earlier zone")
	assert.Equal(t, "Asia/Tokyo", items[1].Timezone)
}

func TestCrontabSource_TZAssignmentFollowedByJobOnSameLine(t *testing.T) {
	// A job sharing a line with the assignment must not be classified as part of it.
	items := extractCrontab(t, "a.cron", `CRON_TZ=Asia/Tokyo 0 2 * * * /x.sh
0 3 * * * /y.sh
`)

	require.Len(t, items, 2, "the job sharing the assignment's line must not be swallowed")
	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "/x.sh", items[0].Command)
	assert.Equal(t, "Asia/Tokyo", items[0].Timezone)
	assert.Equal(t, 1, items[0].Locator.Line,
		"the job shares the line number of the assignment it followed")
	assert.Equal(t, "Asia/Tokyo", items[1].Timezone,
		"a later job must see the real zone, not the swallowed job text")
}

func TestCrontabSource_TZAssignmentWithTrailingCommentIsNotAJob(t *testing.T) {
	items := extractCrontab(t, "a.cron", `CRON_TZ=Asia/Tokyo # tokyo box
0 2 * * * /x.sh
`)

	require.Len(t, items, 1, "a trailing comment on the assignment must not itself become an item")
	assert.Equal(t, "Asia/Tokyo", items[0].Timezone)
}

func TestCrontabSource_NoZoneAssignmentLeavesTimezoneEmpty(t *testing.T) {
	items := extractCrontab(t, "a.cron", "0 2 * * * /usr/bin/backup.sh\n")

	require.Len(t, items, 1)
	assert.Empty(t, items[0].Timezone,
		"no CRON_TZ or TZ assignment means inherit the invocation default")
}

func TestCrontabSource_RejectsACronTZThatDoesNotExist(t *testing.T) {
	// A zone that does not exist must be validated when read and must not poison the jobs that follow it.
	items := extractCrontab(t, "a.cron", `CRON_TZ=Not/AZone
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 2)
	assert.Equal(t, "CRON_TZ=Not/AZone", items[0].Expression)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.Contains(t, items[0].Reason, "Not/AZone")
	assert.Equal(t, 1, items[0].Locator.Line)

	assert.Equal(t, "0 2 * * *", items[1].Expression)
	assert.Empty(t, items[1].Timezone,
		"a bad zone must never become the timezone of the jobs that follow it")
}

func TestCrontabSource_RejectsATZThatDoesNotExist(t *testing.T) {
	items := extractCrontab(t, "a.cron", `TZ=Not/AZone
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 2)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.Contains(t, items[0].Reason, "Not/AZone")
	assert.Empty(t, items[1].Timezone)
}

func TestCrontabSource_BadCronTZDoesNotOverrideAnEarlierGoodOne(t *testing.T) {
	items := extractCrontab(t, "a.cron", `CRON_TZ=Europe/Paris
CRON_TZ=Not/AZone
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 2)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.Equal(t, "Europe/Paris", items[1].Timezone,
		"the earlier valid assignment must survive a later invalid one")
}

func TestCrontabSource_BadCronTZFollowedByJobOnSameLineStillReportsTheJob(t *testing.T) {
	items := extractCrontab(t, "a.cron", "CRON_TZ=Not/AZone 0 2 * * * /x.sh\n")

	require.Len(t, items, 2, "the invalid assignment and the job it shares a line with are both reported")
	assert.Equal(t, "CRON_TZ=Not/AZone", items[0].Expression)
	assert.Equal(t, inventory.StateInvalid, items[0].State)

	assert.Equal(t, "0 2 * * *", items[1].Expression)
	assert.Equal(t, "/x.sh", items[1].Command)
	assert.Empty(t, items[1].Timezone,
		"a job sharing a line with a rejected assignment must not inherit the bad zone")
}

func TestCrontabSource_ZoneValueQuotesAreStripped(t *testing.T) {
	items := extractCrontab(t, "a.cron", `CRON_TZ="Europe/Paris"
0 2 * * * /usr/bin/backup.sh
`)

	require.Len(t, items, 1)
	assert.Equal(t, "Europe/Paris", items[0].Timezone)
}

func TestCrontabSource_QuotedTZAssignmentFollowedByJobOnSameLine(t *testing.T) {
	// The quoted-zone syntax also occurs as a crontab line and must strip the same way.
	items := extractCrontab(t, "a.cron", "CRON_TZ='Asia/Tokyo' 0 2 * * * /x.sh\n")

	require.Len(t, items, 1)
	assert.Equal(t, "0 2 * * *", items[0].Expression)
	assert.Equal(t, "Asia/Tokyo", items[0].Timezone)
}

func TestParseTZAssignment(t *testing.T) {
	cases := []struct {
		name          string
		raw           string
		wantKey       string
		wantValue     string
		wantRemainder string
		wantOK        bool
	}{
		{"cron tz", "CRON_TZ=Europe/Paris", "CRON_TZ", "Europe/Paris", "", true},
		{"tz", "TZ=America/New_York", "TZ", "America/New_York", "", true},
		{"double quoted", `CRON_TZ="Europe/Paris"`, "CRON_TZ", "Europe/Paris", "", true},
		{"single quoted", "TZ='America/New_York'", "TZ", "America/New_York", "", true},
		{"unrelated var", "MAILTO=ops@example.com", "", "", "", false},
		{"no equals", "not an assignment", "", "", "", false},
		{"padded value", "CRON_TZ= Europe/Paris ", "CRON_TZ", "Europe/Paris", "", true},
		{
			"bare assignment", "CRON_TZ=Asia/Tokyo",
			"CRON_TZ", "Asia/Tokyo", "", true,
		},
		{
			"assignment followed by a job", "CRON_TZ=Asia/Tokyo 0 2 * * * /x.sh",
			"CRON_TZ", "Asia/Tokyo", "0 2 * * * /x.sh", true,
		},
		{
			"assignment with a trailing comment", "CRON_TZ=Asia/Tokyo # tokyo box",
			"CRON_TZ", "Asia/Tokyo", "", true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			key, value, remainder, ok := parseTZAssignment(c.raw)
			assert.Equal(t, c.wantOK, ok)
			assert.Equal(t, c.wantKey, key)
			assert.Equal(t, c.wantValue, value)
			assert.Equal(t, c.wantRemainder, remainder)
		})
	}
}

func TestCrontabSource_SkipsNonJobLines(t *testing.T) {
	// Prose can pass crontab.ParseLine's field-count check as a job; only the cron-shape gate keeps it out.
	items := extractCrontab(t, "a.cron", `# a comment
PATH=/usr/bin

not a cron line at all
`)

	assert.Empty(t, items, "comments, env vars, blanks and prose yield no schedules")
}

func TestCrontabSource_CronDIgnoresProseFiles(t *testing.T) {
	// An English README under cron.d must not become a StateInvalid item per line.
	items := extractCrontab(t, "etc/cron.d/README", `Backup jobs live in this directory.
Each fragment follows the standard cron.d layout described in crontab(5).
Contact the platform team before editing anything here.
`)

	assert.Empty(t, items, "a plain-English file under cron.d must not be treated as full of malformed jobs")
}

func TestCrontabSource_CronDIgnoresDateAndDurationLikeProse(t *testing.T) {
	// Both lines start with a digit but neither's first five tokens parse as a schedule.
	items := extractCrontab(t, "etc/cron.d/README", `2024-01-01: we moved the backup job here
5 minutes after boot we run cleanup
`)

	assert.Empty(t, items, "date- and duration-like prose must not be inventoried as jobs")
}

func TestClassifySchedule(t *testing.T) {
	cases := []struct {
		expression     string
		wantValid      bool
		wantReportable bool
	}{
		{"0 2 * * *", true, false},
		{"* * * * *", true, false},
		{"?/5 * * * *", true, false},
		{"@daily", true, false},
		{"99 99 * * *", false, true},
		{"not a cron line at", false, false},
		{"2024-01-01: we moved the backup", false, false},
		{"", false, false},
		// A leading "@" is unambiguously an attempted descriptor, so a parse failure is always reportable.
		{"@dialy", false, true},
		{"@every nonsense", false, true},
		// A bare "@" with nothing attached is ordinary English, not a truncated alias, and not reportable.
		{"@ the team in slack", false, false},
		// A misspelled descriptor as the first field is unambiguous even amid five words of prose after it.
		{"@dialy please tell the team", false, true},
		// A schedule attempt with one unrecognised (not just out-of-range) field is still reportable, not prose.
		{"0 9 * * lun", false, true},
	}
	src := &crontabSource{parser: cronx.NewParser()}
	for _, c := range cases {
		t.Run(c.expression, func(t *testing.T) {
			valid, reportable := src.classifySchedule(c.expression)
			assert.Equal(t, c.wantValid, valid)
			assert.Equal(t, c.wantReportable, reportable)
		})
	}
}

func TestCrontabSource_CronDIsRecognisedRegardlessOfCase(t *testing.T) {
	// Recognising etc/CRON.D as a crontab must also recognise it as system format, or the line parses wrong.
	items := extractCrontab(t, "etc/CRON.D/backup", "0 2 * * * root /usr/bin/backup.sh\n")

	require.Len(t, items, 1)
	assert.Equal(t, "root", items[0].RunAs,
		"a mixed-case cron.d directory must still be parsed as system format")
	assert.Equal(t, "/usr/bin/backup.sh", items[0].Command)
}

func TestCrontabSource_CronDStripsTheUserField(t *testing.T) {
	items := extractCrontab(t, "etc/cron.d/backup", "0 2 * * * root /usr/bin/backup.sh --full\n")

	require.Len(t, items, 1)
	assert.Equal(t, "/usr/bin/backup.sh --full", items[0].Command,
		"cron.d uses the system format: minute hour dom month dow USER command, and arguments after the command survive the split")
	assert.Equal(t, "root", items[0].RunAs, "the user field is recorded, not just discarded")
	assert.Equal(t, inventory.StateActive, items[0].State,
		"a well-formed cron.d line is unaffected")
	assert.Empty(t, items[0].Reason)
}

func TestCrontabSource_CronDAliasRecordsUser(t *testing.T) {
	items := extractCrontab(t, "etc/cron.d/backup", "@daily root /usr/bin/backup.sh --full\n")

	require.Len(t, items, 1)
	assert.Equal(t, "@daily", items[0].Expression)
	assert.Equal(t, "/usr/bin/backup.sh --full", items[0].Command)
	assert.Equal(t, "root", items[0].RunAs)
	assert.Equal(t, inventory.StateActive, items[0].State)
}

func TestCrontabSource_CronDMissingUserFieldWithArgsIsFlaggedNotDestroyed(t *testing.T) {
	// An absolute-path first token is not an account name, so shape-based detection must reject it.
	items := extractCrontab(t, "etc/cron.d/backup", "0 2 * * * /usr/bin/backup.sh --full --dest /mnt/x\n")

	require.Len(t, items, 1, "a malformed job is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State,
		"a cron.d line with no separate user token cannot be trusted to mean the job runs nothing")
	assert.NotEmpty(t, items[0].Reason)
	assert.Equal(t, "/usr/bin/backup.sh --full --dest /mnt/x", items[0].Command,
		"the full original text is kept, not blanked and not partially destroyed")
	assert.Empty(t, items[0].RunAs)
}

func TestCrontabSource_CronDAliasMissingUserFieldWithArgsIsFlaggedNotDestroyed(t *testing.T) {
	items := extractCrontab(t, "etc/cron.d/backup", "@daily /usr/bin/nouser.sh --x\n")

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateInvalid, items[0].State)
	assert.NotEmpty(t, items[0].Reason)
	assert.Equal(t, "/usr/bin/nouser.sh --x", items[0].Command)
	assert.Empty(t, items[0].RunAs)
}

func TestCrontabSource_CronDMissingUserFieldArgLessIsFlaggedNotBlanked(t *testing.T) {
	items := extractCrontab(t, "etc/cron.d/backup", "0 2 * * * /usr/bin/backup.sh\n")

	require.Len(t, items, 1, "a malformed job is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State,
		"a cron.d line with no separate user token cannot be trusted to mean the job runs nothing")
	assert.NotEmpty(t, items[0].Reason)
	assert.Equal(t, "/usr/bin/backup.sh", items[0].Command,
		"the original text is kept, not blanked, so the locator plus command still lead to the line")
	assert.Empty(t, items[0].RunAs)
}

func TestCrontabSource_CronDUserWithNoCommandIsFlaggedNotActive(t *testing.T) {
	// "root" alone looks like a valid user field, but with nothing after it StateActive would be misleading.
	items := extractCrontab(t, "etc/cron.d/backup", "0 2 * * * root\n")

	require.Len(t, items, 1, "a malformed job is still inventoried")
	assert.Equal(t, inventory.StateInvalid, items[0].State,
		"a user field with no command must not be reported as active")
	assert.NotEmpty(t, items[0].Reason)
	assert.Equal(t, "root", items[0].Command,
		"the original text is kept, not blanked, and not misattributed to RunAs")
	assert.Empty(t, items[0].RunAs)
}

func TestSplitSystemUser(t *testing.T) {
	cases := []struct {
		name       string
		command    string
		wantUser   string
		wantRest   string
		wantReason bool
	}{
		{"user and command", "root /usr/bin/backup.sh", "root", "/usr/bin/backup.sh", false},
		{"user, no command", "root", "", "", true},
		{"user, only trailing whitespace", "root   ", "", "", true},
		{"no user, absolute path", "/usr/bin/backup.sh --full", "", "", true},
		{"empty", "", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			user, rest, reason := splitSystemUser(c.command)
			assert.Equal(t, c.wantUser, user)
			assert.Equal(t, c.wantRest, rest)
			if c.wantReason {
				assert.NotEmpty(t, reason)
			} else {
				assert.Empty(t, reason)
			}
		})
	}
}

func TestCrontabSource_UserCrontabKeepsTheWholeCommand(t *testing.T) {
	items := extractCrontab(t, "ops/backup.cron", "0 2 * * * root /usr/bin/backup.sh\n")

	require.Len(t, items, 1)
	assert.Equal(t, "root /usr/bin/backup.sh", items[0].Command,
		"a *.cron file is user format, so a leading word that looks like a user is part of the command")
}

func TestCrontabSource_HandlesLinesOverTheScannerLimit(t *testing.T) {
	// This payload deliberately exceeds bufio.Scanner's 64 KiB token limit.
	payload := strings.Repeat("a", 70*1024)
	content := "0 2 * * * /usr/bin/run.sh " + payload + "\n*/5 * * * * /usr/bin/poll.sh\n"

	items := extractCrontab(t, "a.cron", content)

	require.Len(t, items, 2, "a long line must not abort extraction of the rest of the file")
	assert.Equal(t, "/usr/bin/run.sh "+payload, items[0].Command)
	assert.Equal(t, 1, items[0].Locator.Line)
	assert.Equal(t, "/usr/bin/poll.sh", items[1].Command)
	assert.Equal(t, 2, items[1].Locator.Line, "line numbering must stay correct after the long line")
}

func TestCrontabSource_IgnoresALeadingByteOrderMark(t *testing.T) {
	items := extractCrontab(t, "a.cron", "\xEF\xBB\xBF0 2 * * * /usr/bin/backup.sh\n")

	require.Len(t, items, 1)
	assert.Equal(t, inventory.StateActive, items[0].State,
		"a leading BOM must not glue itself to the minute field and break parsing")
	assert.Equal(t, "0 2 * * *", items[0].Expression)
}

// fakeFileInfo reports a chosen size without a real file of that size, keeping the oversized-file test fast.
type fakeFileInfo struct {
	name string
	size int64
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

func TestCrontabSource_RefusesOversizedFiles(t *testing.T) {
	src := NewCrontabSource()
	fsys := fstest.MapFS{"huge.cron": &fstest.MapFile{Data: []byte("0 2 * * * /usr/bin/backup.sh\n")}}
	unit := Unit{
		Path: "huge.cron",
		Info: fakeFileInfo{name: "huge.cron", size: maxUnitBytes + 1},
	}

	items, err := src.Extract(unit, fsys)

	require.Error(t, err, "a file over the size ceiling must be refused rather than read")
	assert.Nil(t, items)
	assert.Contains(t, err.Error(), "huge.cron")
}

func TestCrontabSource_AllowsFilesAtTheCeiling(t *testing.T) {
	src := NewCrontabSource()
	content := "0 2 * * * /usr/bin/backup.sh\n"
	fsys := fstest.MapFS{"a.cron": &fstest.MapFile{Data: []byte(content)}}
	unit := Unit{
		Path: "a.cron",
		Info: fakeFileInfo{name: "a.cron", size: maxUnitBytes},
	}

	items, err := src.Extract(unit, fsys)

	require.NoError(t, err, "a file exactly at the ceiling is not oversized")
	require.Len(t, items, 1)
}

func TestCrontabSource_Match(t *testing.T) {
	src := NewCrontabSource()

	matches := []string{
		"ops/backup.cron", "crontab", "ops/deploy.crontab",
		"etc/cron.d/backup", "config/cron.d/nightly",
		"ops/BACKUP.CRON", "CRONTAB",
		"etc/CRON.D/backup", "etc/Cron.D/backup",
	}
	for _, p := range matches {
		t.Run(p, func(t *testing.T) {
			assert.True(t, src.Match(Unit{Path: p}))
		})
	}

	rejects := []string{
		"k8s/backup.yaml", "README.md", "cron.go", "docs/crontab.md",
	}
	for _, p := range rejects {
		t.Run(p, func(t *testing.T) {
			assert.False(t, src.Match(Unit{Path: p}))
		})
	}
}

func TestForEachLine(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
		{"trailing newline", "a\nb\n", []string{"a", "b"}},
		{"blank line before eof", "a\nb\n\n", []string{"a", "b", ""}},
		{"single blank line", "\n", []string{""}},
		{"crlf", "a\r\nb\r\n", []string{"a", "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			var lineNumbers []int
			forEachLine([]byte(c.in), func(lineNumber int, line string) {
				lineNumbers = append(lineNumbers, lineNumber)
				got = append(got, line)
			})
			assert.Equal(t, c.want, got)
			for i, n := range lineNumbers {
				assert.Equal(t, i+1, n, "line numbers must be 1-indexed and consecutive")
			}
		})
	}
}

func TestCrontabSource_ID(t *testing.T) {
	assert.Equal(t, "crontab", NewCrontabSource().ID())
}

func TestDefault_IncludesTheCrontabSource(t *testing.T) {
	registry, err := Default()
	require.NoError(t, err)

	var ids []string
	for _, s := range registry.Sources() {
		ids = append(ids, s.ID())
	}
	assert.Contains(t, ids, "crontab")
}
