package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	consensusclient "github.com/attestantio/go-eth2-client"
	apiv1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/rs/zerolog"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, " usage: %s <REST endpoint>\n", os.Args[0])
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := http.New(ctx,
		http.WithLogLevel(zerolog.Disabled),
		http.WithAddress(os.Args[1]),
	)
	if err != nil {
		panic(err)
	}

	if err := client.(consensusclient.EventsProvider).Events(ctx, []string{"block"}, func(event *apiv1.Event) {
		data := event.Data.(*apiv1.BlockEvent)
		block, err := client.(consensusclient.SignedBeaconBlockProvider).SignedBeaconBlock(ctx, fmt.Sprintf("%#x", data.Block))
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to obtan block: %v\n", err)
			return
		}

		var ssz []byte
		switch block.Version {
		case spec.DataVersionPhase0:
			ssz, err = block.Phase0.MarshalSSZ()
		case spec.DataVersionAltair:
			ssz, err = block.Altair.MarshalSSZ()
		case spec.DataVersionBellatrix:
			ssz, err = block.Bellatrix.MarshalSSZ()
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal block: %v\n", err)
			return
		}
		err = os.WriteFile(fmt.Sprintf("%#x.ssz", data.Block), ssz, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to write block: %v\n", err)
		}
	}); err != nil {
		panic(err)
	}

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	for {
		sig := <-sigCh
		if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == os.Interrupt || sig == os.Kill {
			break
		}
	}
}
