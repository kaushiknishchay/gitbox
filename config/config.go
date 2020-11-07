package config

var (
	// PORT on which app will run
	PORT string

	// REPO_BASE_DIR base directory where all repos will reside
	REPO_BASE_DIR string
)

// PerPageCommitCount max number of commits to show on /log at a time
const PerPageCommitCount int64 = 1000

// CommitSeparator used to split git log commits output
const CommitSeparator string = "^^$$^^$$"

// GitLogFormat format in which the git log is output
const GitLogFormat string = `--pretty=format:{"commit": "%H","subject": "%s", "author": {"name": "%aN", "email": "%aE", "date": "%ad"},"commiter": {"name": "%cN", "email": "%cE", "date": "%cd"}}` + CommitSeparator
