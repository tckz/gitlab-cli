package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

var optToken = flag.String("token", "", "private token")
var optGroup = flag.String("group", "", "Group name")
var optURLPrefix = flag.String("url-prefix", "", "URL prefix of gitlab server")
var optTimeout = flag.Duration("timeout", 10*time.Second, "Timeout of http request")

func init() {
	godotenv.Load()
	rep := strings.NewReplacer("-", "_")
	flag.VisitAll(func(f *flag.Flag) {
		if s := os.Getenv("GITLAB_CLI_" + rep.Replace(strings.ToUpper(f.Name))); s != "" {
			f.Value.Set(s)
		}
	})
	flag.Parse()
}

func main() {

	if *optURLPrefix == "" {
		log.Fatalf("--url-prefix must be specified")
	}

	if *optGroup == "" {
		log.Fatalf("--group must be specified")
	}

	if *optToken == "" {
		log.Fatalf("--token must be specified")
	}

	cl := http.DefaultClient

	u, err := url.Parse(*optURLPrefix)
	if err != nil {
		panic(err)
	}
	u.Path = path.Join(u.Path, fmt.Sprintf("/api/v4/groups/%s/projects", *optGroup))

	w := os.Stdout

	for page := 1; ; page++ {
		vals := url.Values{}
		vals.Add("per_page", "100")
		vals.Add("page", strconv.Itoa(page))
		u.RawQuery = vals.Encode()

		log.Printf("url=%s", u.String())

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			panic(err)
		}

		req.Header.Add("Private-Token", *optToken)

		noMorePages, err := func() (bool, error) {
			ctx, cancel := context.WithTimeout(context.Background(), *optTimeout)
			defer cancel()

			req := req.WithContext(ctx)
			res, err := cl.Do(req)
			if err != nil {
				return false, errors.Wrapf(err, "*** client.Do")
			}
			defer func() {
				defer res.Body.Close()
				io.Copy(ioutil.Discard, res.Body)
			}()

			if res.StatusCode != http.StatusOK {
				return false, fmt.Errorf("status=%s", res.Status)
			}

			io.Copy(w, res.Body)
			w.WriteString("\n")

			total := res.Header.Get("x-total-pages")
			return total == strconv.Itoa(page) || total == "", nil
		}()
		if err != nil {
			log.Fatalf("*** %v", err)
		}

		if noMorePages {
			break
		}
	}

}
