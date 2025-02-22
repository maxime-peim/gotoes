package cmd

import (
	"time"

	strava "github.com/maxime-peim/gotoes/pkg"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var timestampCmd = &cobra.Command{
	Use:     "timestamp",
	Aliases: []string{"ts"},
	Short:   "Change the timestamp of a GPX file",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		gpxFile := args[0]
		speed, _ := cmd.Flags().GetUint32("speed")
		startTime, _ := cmd.Flags().GetString("start-time")
		output, _ := cmd.Flags().GetString("output")

		if startTime == "" {
			startTime = time.Now().Format(time.RFC3339)
		}

		params := &strava.AddTimestampsToGPXParams{
			GPXFile:      gpxFile,
			OutputFile:   output,
			DesiredSpeed: speed,
			StartTime:    startTime,
		}
		if err := strava.AddTimestampsToGPX(params); err != nil {
			return errors.Wrap(err, "failed to add timestamps to GPX")
		}
		return nil
	},
}

func init() {
	timestampCmd.Flags().Uint32P("speed", "s", 0, "Desired speed in km/h")
	timestampCmd.Flags().StringP("start-time", "t", "", "Start time in RFC3339 format")
	timestampCmd.Flags().StringP("output", "o", "", "Output file")
	timestampCmd.MarkFlagRequired("speed")
}
