package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"fleep-ci-notifications/Godeps/_workspace/src/github.com/go-martini/martini"
)

type circleCIData struct {
	Payload struct {
		BuildURL       string `json:"build_url"`
		BuildNum       int    `json:"build_num"`
		Branch         string `json:"branch"`
		RepositoryName string `json:"reponame"`
		CommiterName   string `json:"committer_name"`
		Outcome        string `json:"outcome"`
	} `json:"payload"`
}

type buildKiteData struct {
	Build struct {
		ID      string      `json:"id"`
		URL     string      `json:"url"`
		WebURL  string      `json:"web_url"`
		Number  int         `json:"number"`
		State   string      `json:"state"`
		Message string      `json:"message"`
		Commit  string      `json:"commit"`
		Branch  string      `json:"branch"`
		Tag     interface{} `json:"tag"`
		Source  string      `json:"source"`
		Creator struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
			CreatedAt string `json:"created_at"`
		} `json:"creator"`
		CreatedAt   string `json:"created_at"`
		ScheduledAt string `json:"scheduled_at"`
		StartedAt   string `json:"started_at"`
		FinishedAt  string `json:"finished_at"`
		MetaData    struct {
			BuildkiteGitBranch string `json:"buildkite:git:branch"`
			BuildkiteGitCommit string `json:"buildkite:git:commit"`
		} `json:"meta_data"`
	} `json:"build"`
	Pipeline struct {
		ID         string `json:"id"`
		URL        string `json:"url"`
		WebURL     string `json:"web_url"`
		Name       string `json:"name"`
		Slug       string `json:"slug"`
		Repository string `json:"repository"`
		Provider   struct {
			ID       string `json:"id"`
			Settings struct {
				BuildPullRequests          bool   `json:"build_pull_requests"`
				BuildTags                  bool   `json:"build_tags"`
				PublishCommitStatus        bool   `json:"publish_commit_status"`
				PublishCommitStatusPerStep bool   `json:"publish_commit_status_per_step"`
				Repository                 string `json:"repository"`
			} `json:"settings"`
			WebhookURL string `json:"webhook_url"`
		} `json:"provider"`
		BuildsURL string `json:"builds_url"`
		CreatedAt string `json:"created_at"`
		Steps     []struct {
			Type                string `json:"type"`
			Name                string `json:"name"`
			Command             string `json:"command"`
			ArtifactPaths       string `json:"artifact_paths"`
			BranchConfiguration string `json:"branch_configuration"`
			Env                 struct {
			} `json:"env"`
			TimeoutInMinutes interface{}   `json:"timeout_in_minutes"`
			AgentQueryRules  []interface{} `json:"agent_query_rules"`
			Concurrency      interface{}   `json:"concurrency"`
			Parallelism      interface{}   `json:"parallelism"`
		} `json:"steps"`
		Env struct {
			NPMAUTHTOKEN string `json:"NPM_AUTH_TOKEN"`
			TUTUMAUTH    string `json:"TUTUM_AUTH"`
		} `json:"env"`
		ScheduledBuildsCount int `json:"scheduled_builds_count"`
		RunningBuildsCount   int `json:"running_builds_count"`
		ScheduledJobsCount   int `json:"scheduled_jobs_count"`
		RunningJobsCount     int `json:"running_jobs_count"`
		WaitingJobsCount     int `json:"waiting_jobs_count"`
	} `json:"pipeline"`
	Sender struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"sender"`
}

var circleCITemplate = `Build number *%d* based on a commit by *%s* to a *%s* repository branch *%s* completed on CircleCI with a status of *%s*.
More information is available on %s<<CircleCI>>.`

var buildKiteTemplate = `Build number *%d* based on a commit by *%s* to a *%s* repository branch *%s* completed on BuildKite with a status of *%s*.
Additional build information is available on %s<<BuildKite>>. Details on the commit are available on %s<<%s>>.`

func main() {
	m := martini.Classic()

	m.Get("/", func() (int, string) {
		return 200, "I'm alive."
	})

	m.Post("/", func() (int, string) {
		return 200, "I'm alive."
	})

	m.Post("/webhook/:auth/:hash", func(params martini.Params, req *http.Request) (int, string) {
		var message string

		defer req.Body.Close()
		j := json.NewDecoder(req.Body)

		if req.UserAgent() == "Buildkite-Request" {
			var buildkite buildKiteData
			err := j.Decode(&buildkite)
			if err != nil {
				return 500, "Bad request."
			}

			var providerURL string
			if buildkite.Pipeline.Provider.ID == "bitbucket" {
				providerURL = "https://bitbucket.org/" + buildkite.Pipeline.Provider.Settings.Repository + "/commits/" + buildkite.Build.Commit
			} else if buildkite.Pipeline.Provider.ID == "github" {
				providerURL = "https://github.com/" + buildkite.Pipeline.Provider.Settings.Repository + "/commits/" + buildkite.Build.Commit
			}

			r := regexp.MustCompile("Author:.{1,5}(?P<author>.+ <.+>)\\nA")
			message = fmt.Sprintf(buildKiteTemplate,
				buildkite.Build.Number,
				r.FindStringSubmatch(buildkite.Build.MetaData.BuildkiteGitCommit)[1],
				buildkite.Pipeline.Repository,
				buildkite.Build.Branch,
				buildkite.Build.State,
				buildkite.Build.WebURL,
				providerURL,
				strings.Title(buildkite.Pipeline.Provider.ID))
		} else {
			var circle circleCIData
			err := j.Decode(&circle)
			if err != nil {
				return 500, "Bad request."
			}

			message = fmt.Sprintf(circleCITemplate,
				circle.Payload.BuildNum,
				circle.Payload.CommiterName,
				circle.Payload.RepositoryName,
				circle.Payload.Branch,
				circle.Payload.Outcome,
				circle.Payload.BuildURL)
		}

		if os.Getenv("WEBHOOK_SECRET") == params["auth"] {
			fleepURL := "https://fleep.io/hook/" + params["hash"]

			resp, err := http.PostForm(fleepURL, url.Values{"message": {string(message)}})
			if err != nil {
				fmt.Println("Error during POST request to Fleep", err)
				return 500, "Error calling Fleep Hook"
			}

			defer resp.Body.Close()
			fmt.Println("Fleep hook response status:", resp.Status)

			return 200, "Successfully proxied hook " + params["hash"]
		}

		return 401, "Unauthorized."
	})

	m.Run()
}
