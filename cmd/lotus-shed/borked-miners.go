package main

import (
	"fmt"

	"github.com/filecoin-project/go-state-types/abi"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	"github.com/urfave/cli/v2"
)

var borkedMinersCmd = &cli.Command{
	Name:        "borked-miners",
	Description: "information about 'borked' miner actors with late or frozen deadline crons",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "tipset",
			Usage: "specify tipset state to search on (pass comma separated array of cids)",
		},
		&cli.BoolFlag{
			Name:  "future",
			Usage: "print info of miners with last deadline cron in the future (normal for v0 and early v2 actors)",
		},
	},
	Action: func(c *cli.Context) error {
		api, acloser, err := lcli.GetFullNodeAPI(c)
		if err != nil {
			return err
		}
		defer acloser()
		ctx := lcli.ReqContext(c)

		ts, err := lcli.LoadTipSet(ctx, c, api)
		if err != nil {
			return err
		}
		if ts == nil {
			ts, err = api.ChainHead(ctx)
			if err != nil {
				return err
			}
		}

		queryEpoch := ts.Height()

		mAddrs, err := api.StateListMiners(ctx, ts.Key())
		if err != nil {
			return err
		}

		for _, mAddr := range mAddrs {
			st, err := api.StateReadState(ctx, mAddr, ts.Key())
			if err != nil {
				return err
			}
			minerSt, ok := st.State.(miner.State)
			if !ok {
				fmt.Printf("bad minerSt type %T: %v\n", minerSt, minerSt)
				continue
				//				return xerrors.Errorf("internal error: reading miner address returns non-miner state")
			}
			pps := minerSt.ProvingPeriodStart
			dlIdx := minerSt.CurrentDeadline
			latestDeadline := pps + abi.ChainEpoch(int64(dlIdx))*miner.WPoStChallengeWindow
			nextDeadline := latestDeadline + miner.WPoStChallengeWindow

			// Need +1 because last epoch of the deadline queryEpoch = x + 59 cron gets run and
			// state is left with latestDeadline = x + 60
			if c.Bool("future") && latestDeadline > queryEpoch+1 {
				fmt.Printf("%s -- last deadline start in future epoch %d > query epoch %d + 1\n", mAddr, latestDeadline, queryEpoch)
			}

			// Equality is an error because last epoch of the deadline queryEpoch = x + 59.  Cron
			// should get run and bump latestDeadline = x + 60 so nextDeadline = x + 120
			if queryEpoch >= nextDeadline {
				fmt.Printf("%x -- next deadline start in non-future epoch %d <= query epoch %d\n", mAddr, nextDeadline, queryEpoch)
			}

		}

		return nil
	},
}
