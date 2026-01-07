package diff

import (
	"bytes"
	"testing"

	"github.com/hzerrad/cronkit/internal/crontab"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextRenderer_Render(t *testing.T) {
	diff := &Diff{
		Added: []Change{
			{
				Type: ChangeTypeAdded,
				NewJob: &crontab.Job{
					Expression: "*/15 * * * *",
					Command:    "/usr/bin/check.sh",
					Comment:    "New job",
				},
			},
		},
		Removed: []Change{
			{
				Type: ChangeTypeRemoved,
				OldJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/old.sh",
				},
			},
		},
		Modified: []Change{
			{
				Type: ChangeTypeModified,
				OldJob: &crontab.Job{
					Expression: "0 3 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "Old comment",
				},
				NewJob: &crontab.Job{
					Expression: "0 3 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "New comment",
				},
				FieldsChanged: []string{"comment"},
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Added Jobs")
	assert.Contains(t, output, "*/15 * * * *")
	assert.Contains(t, output, "Removed Jobs")
	assert.Contains(t, output, "0 2 * * *")
	assert.Contains(t, output, "Modified Jobs")
	assert.Contains(t, output, "comment")
}

func TestJSONRenderer_Render(t *testing.T) {
	diff := &Diff{
		Added: []Change{
			{
				Type: ChangeTypeAdded,
				NewJob: &crontab.Job{
					Expression: "*/15 * * * *",
					Command:    "/usr/bin/check.sh",
					LineNumber: 1,
				},
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"added"`)
	assert.Contains(t, output, `"*/15 * * * *"`)
	assert.Contains(t, output, `"type"`)
	assert.Contains(t, output, `"added"`)
	assert.Contains(t, output, `"summary"`)
}

func TestUnifiedRenderer_Render(t *testing.T) {
	diff := &Diff{
		Added: []Change{
			{
				Type: ChangeTypeAdded,
				NewJob: &crontab.Job{
					Expression: "*/15 * * * *",
					Command:    "/usr/bin/check.sh",
				},
			},
		},
		Removed: []Change{
			{
				Type: ChangeTypeRemoved,
				OldJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/old.sh",
				},
			},
		},
	}

	renderer := &UnifiedRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "--- old crontab")
	assert.Contains(t, output, "+++ new crontab")
	assert.Contains(t, output, "-0 2 * * *")
	assert.Contains(t, output, "+*/15 * * * *")
}

func TestNewRenderer(t *testing.T) {
	t.Run("text format", func(t *testing.T) {
		renderer, err := NewRenderer("text")
		require.NoError(t, err)
		assert.IsType(t, &TextRenderer{}, renderer)
	})

	t.Run("json format", func(t *testing.T) {
		renderer, err := NewRenderer("json")
		require.NoError(t, err)
		assert.IsType(t, &JSONRenderer{}, renderer)
	})

	t.Run("unified format", func(t *testing.T) {
		renderer, err := NewRenderer("unified")
		require.NoError(t, err)
		assert.IsType(t, &UnifiedRenderer{}, renderer)
	})

	t.Run("default format", func(t *testing.T) {
		renderer, err := NewRenderer("")
		require.NoError(t, err)
		assert.IsType(t, &TextRenderer{}, renderer)
	})

	t.Run("invalid format", func(t *testing.T) {
		_, err := NewRenderer("invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown format")
	})
}

func TestTextRenderer_ShowUnchanged(t *testing.T) {
	diff := &Diff{
		Unchanged: []Change{
			{
				Type: ChangeTypeUnchanged,
				NewJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
				},
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	options := &RenderOptions{ShowUnchanged: true}
	err := renderer.Render(&buf, diff, options)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Unchanged Jobs")
	assert.Contains(t, output, "0 2 * * *")
}

func TestTextRenderer_IgnoreOptions(t *testing.T) {
	diff := &Diff{
		EnvChanges: []EnvChange{
			{
				Type:     ChangeTypeAdded,
				Key:      "PATH",
				NewValue: "/usr/bin",
			},
		},
		CommentChanges: []CommentChange{
			{
				Type:    ChangeTypeAdded,
				NewLine: "# New comment",
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	options := &RenderOptions{
		IgnoreEnv:      true,
		IgnoreComments: true,
	}
	err := renderer.Render(&buf, diff, options)

	require.NoError(t, err)
	output := buf.String()

	assert.NotContains(t, output, "Environment Variable Changes")
	assert.NotContains(t, output, "Comment Changes")
}

func TestTextRenderer_NoChanges(t *testing.T) {
	diff := &Diff{}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "No changes detected")
}

func TestJSONRenderer_ShowUnchanged(t *testing.T) {
	diff := &Diff{
		Unchanged: []Change{
			{
				Type: ChangeTypeUnchanged,
				NewJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
				},
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	options := &RenderOptions{ShowUnchanged: true}
	err := renderer.Render(&buf, diff, options)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"unchanged"`)
}

func TestJSONRenderer_IgnoreOptions(t *testing.T) {
	diff := &Diff{
		EnvChanges: []EnvChange{
			{
				Type:     ChangeTypeAdded,
				Key:      "PATH",
				NewValue: "/usr/bin",
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	options := &RenderOptions{IgnoreEnv: true}
	err := renderer.Render(&buf, diff, options)

	require.NoError(t, err)
	output := buf.String()

	// When IgnoreEnv is true, envChanges should not be included in output
	assert.NotContains(t, output, `"envChanges"`)
	assert.NotContains(t, output, `"PATH"`)
}

func TestUnifiedRenderer_AllChangeTypes(t *testing.T) {
	diff := &Diff{
		Added: []Change{
			{
				Type: ChangeTypeAdded,
				NewJob: &crontab.Job{
					Expression: "*/5 * * * *",
					Command:    "/usr/bin/frequent.sh",
					Comment:    "New job",
				},
			},
		},
		Removed: []Change{
			{
				Type: ChangeTypeRemoved,
				OldJob: &crontab.Job{
					Expression: "0 1 * * *",
					Command:    "/usr/bin/removed.sh",
					Comment:    "Old job",
				},
			},
		},
		Modified: []Change{
			{
				Type: ChangeTypeModified,
				OldJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "old",
				},
				NewJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "new",
				},
			},
		},
	}

	renderer := &UnifiedRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "--- old crontab")
	assert.Contains(t, output, "+++ new crontab")
	assert.Contains(t, output, "-0 1 * * *")
	assert.Contains(t, output, "+*/5 * * * *")
}

func TestTextRenderer_EnvVarChanges(t *testing.T) {
	diff := &Diff{
		EnvChanges: []EnvChange{
			{
				Type:     ChangeTypeAdded,
				Key:      "TZ",
				NewValue: "UTC",
			},
			{
				Type:     ChangeTypeRemoved,
				Key:      "PATH",
				OldValue: "/usr/bin",
			},
			{
				Type:     ChangeTypeModified,
				Key:      "HOME",
				OldValue: "/old/home",
				NewValue: "/new/home",
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Environment Variable Changes")
	assert.Contains(t, output, "TZ")
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "HOME")
}

func TestTextRenderer_CommentChanges(t *testing.T) {
	diff := &Diff{
		CommentChanges: []CommentChange{
			{
				Type:    ChangeTypeAdded,
				NewLine: "# New comment",
			},
			{
				Type:    ChangeTypeRemoved,
				OldLine: "# Old comment",
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Comment Changes")
	assert.Contains(t, output, "New comment")
	assert.Contains(t, output, "Old comment")
}

func TestTextRenderer_ModifiedJobWithAllFields(t *testing.T) {
	diff := &Diff{
		Modified: []Change{
			{
				Type: ChangeTypeModified,
				OldJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/old.sh",
					Comment:    "old comment",
				},
				NewJob: &crontab.Job{
					Expression: "0 3 * * *",
					Command:    "/usr/bin/new.sh",
					Comment:    "new comment",
				},
				FieldsChanged: []string{"expression", "command", "comment"},
			},
		},
	}

	renderer := &TextRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Modified Jobs")
	assert.Contains(t, output, "expression")
	assert.Contains(t, output, "command")
	assert.Contains(t, output, "comment")
	assert.Contains(t, output, "Old expression")
	assert.Contains(t, output, "New expression")
}

func TestJSONRenderer_ModifiedJob(t *testing.T) {
	diff := &Diff{
		Modified: []Change{
			{
				Type: ChangeTypeModified,
				OldJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "old",
					LineNumber: 1,
				},
				NewJob: &crontab.Job{
					Expression: "0 2 * * *",
					Command:    "/usr/bin/backup.sh",
					Comment:    "new",
					LineNumber: 1,
				},
				FieldsChanged: []string{"comment"},
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"modified"`)
	assert.Contains(t, output, `"fieldsChanged"`)
	assert.Contains(t, output, `"comment"`)
	assert.Contains(t, output, `"oldComment"`)
}

func TestJSONRenderer_CommentChanges(t *testing.T) {
	diff := &Diff{
		CommentChanges: []CommentChange{
			{
				Type:    ChangeTypeAdded,
				NewLine: "# Added comment",
			},
			{
				Type:    ChangeTypeRemoved,
				OldLine: "# Removed comment",
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	options := &RenderOptions{IgnoreComments: false}
	err := renderer.Render(&buf, diff, options)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"commentChanges"`)
	assert.Contains(t, output, `"added"`)
	assert.Contains(t, output, `"removed"`)
}

func TestJSONRenderer_EnvVarAllTypes(t *testing.T) {
	diff := &Diff{
		EnvChanges: []EnvChange{
			{
				Type:     ChangeTypeAdded,
				Key:      "TZ",
				NewValue: "UTC",
			},
			{
				Type:     ChangeTypeRemoved,
				Key:      "PATH",
				OldValue: "/usr/bin",
			},
			{
				Type:     ChangeTypeModified,
				Key:      "HOME",
				OldValue: "/old",
				NewValue: "/new",
			},
		},
	}

	renderer := &JSONRenderer{}
	var buf bytes.Buffer
	err := renderer.Render(&buf, diff, nil)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, `"envChanges"`)
	assert.Contains(t, output, `"TZ"`)
	assert.Contains(t, output, `"PATH"`)
	assert.Contains(t, output, `"HOME"`)
}
