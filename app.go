package main

import "os"
import "fmt"
import "net/http"
import "net/url"
import "github.com/go-martini/martini"
import "github.com/martini-contrib/binding"

type CIData struct {
  Payload struct {
    BuildUrl string `json:"build_url"`
    BuildNum int `json:"build_num"`
    Branch string `json:"branch"`
    CommiterName string `json:"committer_name"`
    Outcome string `json:"outcome"`
  } `json:"payload"`
}

var template = `Build number *%d* based on a commit by *%s* completed on CircleCI with a status of *%s*.
More information is available on %s<<CircleCI>>.`

func main() {
    m := martini.Classic()

    m.Post("/webhook/:auth/:hash", binding.Bind(CIData{}), func(params martini.Params, ci CIData) (int, string) {
      if os.Getenv("WEBHOOK_SECRET") == params["auth"] {
        fleep_url := "https://fleep.io/hook/" + params["hash"]

        message := fmt.Sprintf(template, ci.Payload.BuildNum, ci.Payload.CommiterName, 
          ci.Payload.Outcome, ci.Payload.BuildUrl)

        resp, err := http.PostForm(fleep_url, url.Values{"message": {string(message)}})
        if err != nil {
         fmt.Println("Error during POST request to Fleep", err)
         return 500, "Error calling Fleep Hook"
        }

        defer resp.Body.Close()
        fmt.Println("Fleep hook response status:", resp.Status)

        return 200, "Successfully proxied hook " + params["hash"]
      } else {
        return 401, "Unauthorized."
      }
    })

    m.Run()
}