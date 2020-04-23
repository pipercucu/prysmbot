package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/wealdtech/go-bytesutil"
)

func getBlockCommandResult(command string, parameters []string) string {
		if len(parameters) != 1 {
			log.Error("Expected 1 parameter for validator command")
		}
		reqSlot, err := strconv.Atoi(parameters[0])
		if err != nil {
			log.WithError(err).Error(err, "failed to convert")
			os.Exit(1)
		}
		req := &eth.ListBlocksRequest{
			QueryFilter: &eth.ListBlocksRequest_Slot{
				Slot: uint64(reqSlot),
			},
		}
		blocks, err := beaconClient.ListBlocks(context.Background(), req)
		if err != nil {
			log.WithError(err).Error(err, "failed to get committees")
			os.Exit(1)
		}
		if len(blocks.BlockContainers) < 1 {
			return fmt.Sprintf("No block found for slot %d", reqSlot)
		}
		block := blocks.BlockContainers[0].Block
		switch command {
		case blockGraffiti.command, blockGraffiti.shorthand:
			graffiti := block.Block.Body.Graffiti
			emptyGraffiti := bytesutil.ToBytes32([]byte{0})
			if bytes.Equal(graffiti, emptyGraffiti[:]) {
				return fmt.Sprintf("Graffiti for block %d is empty", reqSlot)
			}
			fmt.Printf("%s\n", graffiti)
			return fmt.Sprintf(blockGraffiti.responseText, reqSlot, graffiti)
		case blockProposer.command, blockProposer.shorthand:
			return fmt.Sprintf(blockProposer.responseText, reqSlot, block.Block.ProposerIndex)
		}
	return ""
}