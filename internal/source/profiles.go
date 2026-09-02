package source

// K8sCronJob describes a Kubernetes CronJob manifest. Note timeZone's capital
// Z, which differs from Argo's spelling.
var K8sCronJob = Profile{
	ID:          "k8s",
	Extensions:  []string{".yaml", ".yml"},
	Match:       []FieldMatch{{Path: "kind", Equals: "CronJob"}},
	Schedules:   []Path{"spec.schedule"},
	Timezone:    "spec.timeZone",
	Suspend:     "spec.suspend",
	Concurrency: "spec.concurrencyPolicy",
	Command:     "spec.jobTemplate.spec.template.spec.containers[0].image",
	Dialect:     "vixie",
}

// ArgoCronWorkflow describes an Argo CronWorkflow; both spec.schedules and spec.schedule are tried.
var ArgoCronWorkflow = Profile{
	ID:          "argo",
	Extensions:  []string{".yaml", ".yml"},
	Match:       []FieldMatch{{Path: "kind", Equals: "CronWorkflow"}},
	Schedules:   []Path{"spec.schedules[]", "spec.schedule"},
	Timezone:    "spec.timezone",
	Suspend:     "spec.suspend",
	Concurrency: "spec.concurrencyPolicy",
	Dialect:     "vixie",
}

// GitHubActions describes a workflow's schedule triggers; GitHub always runs these in UTC.
var GitHubActions = Profile{
	ID:            "gha",
	Extensions:    []string{".yaml", ".yml"},
	DirPrefix:     ".github/workflows",
	Schedules:     []Path{"on.schedule[].cron"},
	TimezoneFixed: "UTC",
	Dialect:       "vixie",
}

// Default returns a registry of the built-in sources; MatchAll runs every source that recognises a path.
func Default() (*Registry, error) {
	sources := []Source{NewCrontabSource()}
	for _, p := range []Profile{K8sCronJob, ArgoCronWorkflow, GitHubActions} {
		src, err := NewProfileSource(p)
		if err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	return NewRegistry(sources...), nil
}
