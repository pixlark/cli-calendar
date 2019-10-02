package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/api/calendar/v3"
)

var cfgDir string

func min(a, b int) int { if a < b { return a } else { return b } }
func max(a, b int) int { if a > b { return a } else { return b } }

func getClient(config *oauth2.Config) *http.Client {
	tokFile := cfgDir + "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

type Terminal struct {
	width, height int
}

func newTerminal() Terminal {
	width, height, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		log.Fatal("Error initializing terminal")
	}
	if width <= 6 {
		log.Fatal("Terminal too narrow")
	}
	return Terminal{width, height}
}

func (t *Terminal) horizontal() {
	fmt.Printf("|")
	for i := 0; i < t.width - 2; i++ {
		fmt.Printf("-");
	}
	fmt.Printf("|")
}

func (t *Terminal) line(str string, args ...interface{}) {
	s := fmt.Sprintf(str, args...)
	w := t.width - 4

	for i := 0; i < len(s); i += w {
		fmt.Printf("| ")
		fmt.Printf("%s", s[i:min(len(s), i+w)])
		for j := 0; j < w - len(s[i:]); j++ {
			fmt.Printf(" ")
		}
		fmt.Print(" |")
	}
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	var vec []string
	for i := 0; i < n - len(s); i++ {
		vec = append(vec, " ")
	}
	return s + strings.Join(vec, "")
}

func outputEvents(events *calendar.Events) {
	term := newTerminal()
	term.horizontal()
	
	for _, item := range events.Items {
		term.line("%s", item.Summary)
		var t time.Time
		err := t.UnmarshalText([]byte(item.Start.DateTime))
		if err != nil {
			log.Fatal("Error with datetime recieved from google API")
		}
		h, m, _ := t.Clock()
		weekday := fmt.Sprintf("%v", t.Weekday())
		term.line("    %s %02d:%02d", pad(weekday, 10), h, m)
		term.horizontal()
		/*
		date := item.Start.DateTime
		if date == "" {
			date = item.Start.Date
		}*/
		//fmt.Printf("%v (%v)\n", item.Summary, date)
	}
}

func main() {
	home := os.Getenv("HOME")
	cfgDir = home + "/.config/calendar/"
	
	b, err := ioutil.ReadFile(cfgDir + "credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}

	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	outputEvents(events)
}
