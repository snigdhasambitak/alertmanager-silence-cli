package alertmanager_silence_cli

import (
	"github.com/snigdhasambitak/alertmanager-silence-cli/cmd"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	appgroup  string
	app       string
	version   string = "1.0"
	branch    string
	revision  string
	buildDate string
)
var (
	mode = kingpin.Flag("mode", "work mode: create/delete/show silence").Default("show").Short('m').String()
	silencePeriod = kingpin.Flag("silence-period", "default period for silenced alerts in hours").Default("2").Int()
	labels = kingpin.Flag("labels", "comma separated silence matching labels, eg. key1=value1,key2=value2").Short('l').String()
	creator = kingpin.Flag("creator", "creator of the silence").Default("auto-silencer").Short('c').String()
	comment = kingpin.Flag("comment", "comment attached to the silence. Recommended to add the jira ticket").Default("auto-silencer").Short('C').String()
	amURL = kingpin.Flag("URL", "Alertmanager URL").Default("http://127.0.0.1").Short('u').String()
	timeout = kingpin.Flag("timeout", "Alertmanager connection timeout").Default("3").Short('t').Int()
)

func main() {
	// parse command line parameters
	kingpin.Parse()

	if err := cmd.Run(*amURL, *timeout, *mode, *labels, *silencePeriod, *creator, *comment); err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	}
}
