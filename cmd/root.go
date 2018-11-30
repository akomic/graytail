package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"graytail/logs"
	"net/url"
	"os"
	"strings"
)

var (
	GTCmd = &cobra.Command{
		Use:   "graytail",
		Short: "",
		Long:  ``,

		Example: `
graytail --uri ws://MyToken@127.0.0.1:20221 -f container_name=nginx

# All fields in json
graytail --uri ws://MyToken@127.0.0.1:20221 --raw-output | jq '.'

# Show custom fields
graytail --uri ws://MyToken@127.0.0.1:20221 --raw-output | jq -r '. | "\(.host) \(.container_name) \(.container_image) \(.short_message)"'`,

		Run: logsRun,
	}
)

func init() {
	GTCmd.Flags().StringP("uri", "", "", "WS URI (e.g. ws://MyToken@127.0.0.1:20221)")
	GTCmd.Flags().StringSliceP("filter", "f", nil, "Filters")
	GTCmd.Flags().BoolP("raw-output", "", false, "Dump complete messages as json")
	GTCmd.Flags().BoolP("local-time", "", false, "Show Local Time")
	GTCmd.Flags().BoolP("no-color", "", false, "Don't use colors")
	GTCmd.Flags().BoolP("verbose", "v", false, "Verbose")
	GTCmd.Flags().BoolP("debug", "d", false, "Debug")

	viper.BindPFlag("uri", GTCmd.Flags().Lookup("uri"))
	viper.BindPFlag("filter", GTCmd.Flags().Lookup("filter"))
	viper.BindPFlag("raw-output", GTCmd.Flags().Lookup("raw-output"))
	viper.BindPFlag("local-time", GTCmd.Flags().Lookup("local-time"))
	viper.BindPFlag("no-color", GTCmd.Flags().Lookup("no-color"))
	viper.BindPFlag("verbose", GTCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("debug", GTCmd.Flags().Lookup("debug"))
}

func logsRun(ccmd *cobra.Command, args []string) {
	uri := viper.GetString("uri")
	filters := viper.GetStringSlice("filter")

	if uri == "" {
		fmt.Println("You need to specify at least URI")
		ccmd.HelpFunc()(ccmd, args)
		os.Exit(1)
	}

	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	queryString := []string{}
	if u.User.Username() != "" {
		queryString = append(queryString, "token="+u.User.Username())
		u.User = nil
	}
	if u.Path == "" {
		u.Path = "/"
	}

	queryString = append(queryString, filters...)
	u.RawQuery = strings.Join(queryString, "&")

	fmt.Println(u.String())

	l := logs.NewLogs(u.String())
	l.Tail()
}
