/*
 * Copyright (C) 2021 Zilliqa
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package core

import (
	"math/big"
	"strconv"

	"github.com/golang/protobuf/proto"

	"github.com/renlulu/gozilliqa-sdklegacy/protobuf"
	"github.com/renlulu/gozilliqa-sdklegacy/util"
)

// https://github.com/Zilliqa/Zilliqa/blob/04162ef0c3c1787ebbd940b7abd6b7ff1d4714ed/src/libData/BlockData/BlockHeader/DSBlockHeader.h
type DsBlockHeader struct {
	BlockHeaderBase BlockHeaderBase
	DsDifficulty    uint32
	Difficulty      uint32
	// The one who proposed this DS block
	// base16 string
	LeaderPubKey string
	// Block index, starting from 0 in the genesis block
	BlockNum uint64
	// Tx Epoch Num then the DS block was generated
	EpochNum uint64
	GasPrice string
	SwInfo   SWInfo
	// key is (base16) public key
	PoWDSWinners map[string]Peer
	// (base16) public key
	RemoveDSNodePubKeys []string
	// todo concrete data type
	DSBlockHashSet     DSBlockHashSet
	GovDSShardVotesMap map[uint32]Pair
}

func NewDsBlockHeaderFromDsBlockT(dst *DsBlockT) *DsBlockHeader {
	dsBlockHeader := &DsBlockHeader{}
	dsBlockHeader.DsDifficulty = dst.Header.DifficultyDS
	dsBlockHeader.Difficulty = dst.Header.Difficulty
	dsBlockHeader.LeaderPubKey = dst.Header.LeaderPubKey

	blockNum, _ := strconv.ParseUint(dst.Header.BlockNum, 10, 64)
	dsBlockHeader.BlockNum = blockNum

	epochNum, _ := strconv.ParseUint(dst.Header.EpochNum, 10, 64)
	dsBlockHeader.EpochNum = epochNum

	dsBlockHeader.GasPrice = dst.Header.GasPrice

	zilliqaUpgradeDS, _ := strconv.ParseUint(dst.Header.SWInfo.Zilliqa[3].(string), 10, 64)
	scillaUpgradeDS, _ := strconv.ParseUint(dst.Header.SWInfo.Scilla[3].(string), 10, 64)

	dsBlockHeader.SwInfo = SWInfo{
		ZilliqaMajorVersion: uint32(dst.Header.SWInfo.Zilliqa[0].(float64)),
		ZilliqaMinorVersion: uint32(dst.Header.SWInfo.Zilliqa[1].(float64)),
		ZilliqaFixVersion:   uint32(dst.Header.SWInfo.Zilliqa[2].(float64)),
		ZilliqaUpgradeDS:    zilliqaUpgradeDS,
		ZilliqaCommit:       uint32(dst.Header.SWInfo.Zilliqa[4].(float64)),
		ScillaMajorVersion:  uint32(dst.Header.SWInfo.Scilla[0].(float64)),
		ScillaMinorVersion:  uint32(dst.Header.SWInfo.Scilla[1].(float64)),
		ScillaFixVersion:    uint32(dst.Header.SWInfo.Scilla[2].(float64)),
		ScillaUpgradeDS:     scillaUpgradeDS,
		ScillaCommit:        uint32(dst.Header.SWInfo.Scilla[4].(float64)),
	}

	winnermap := make(map[string]Peer, len(dst.Header.PoWWinners))
	for i := 0; i < len(dst.Header.PoWWinners); i++ {
		ip := dst.Header.PoWWinnersIP[i].IP
		port := dst.Header.PoWWinnersIP[i].Port

		IPAddress := IP2Long(ip)

		peer := Peer{
			IpAddress:      new(big.Int).SetUint64(uint64(IPAddress)),
			ListenPortHost: port,
		}
		winnermap[dst.Header.PoWWinners[i]] = peer
	}

	dsBlockHeader.PoWDSWinners = winnermap

	var removeDSNodePubKeys []string
	for _, key := range dst.Header.MembersEjected {
		removeDSNodePubKeys = append(removeDSNodePubKeys, key)
	}
	dsBlockHeader.RemoveDSNodePubKeys = removeDSNodePubKeys

	var dsHashSet DSBlockHashSet
	dsHashSet.ShardingHash = util.DecodeHex(dst.Header.ShardingHash)
	dsBlockHeader.DSBlockHashSet = dsHashSet

	governance := make(map[uint32]Pair, 0)
	govs := dst.Header.Governance
	for _, gov := range govs {
		proposalId := gov.ProposalId
		dsmap := make(map[uint32]uint32, 0)
		dsvotes := gov.DSVotes
		for _, dsvote := range dsvotes {
			dsmap[dsvote.VoteValue] = dsvote.VoteCount
		}

		shardmap := make(map[uint32]uint32, 0)
		shardvotes := gov.ShardVotes
		for _, shardvote := range shardvotes {
			shardmap[shardvote.VoteValue] = shardvote.VoteCount
		}

		pair := Pair{
			First:  dsmap,
			Second: shardmap,
		}
		governance[proposalId] = pair
	}

	dsBlockHeader.GovDSShardVotesMap = governance

	dsBlockHeader.BlockHeaderBase.Version = dst.Header.Version
	ch := util.DecodeHex(dst.Header.CommitteeHash)
	var commitHash [32]byte
	copy(commitHash[:], ch)
	dsBlockHeader.BlockHeaderBase.CommitteeHash = commitHash

	ph := util.DecodeHex(dst.Header.PrevHash)
	var prevHash [32]byte
	copy(prevHash[:], ph)
	dsBlockHeader.BlockHeaderBase.PrevHash = prevHash

	return dsBlockHeader
}

func (d *DsBlockHeader) Serialize() []byte {
	h := d.ToProtobuf(false)
	bytes, _ := proto.Marshal(h)
	return bytes
}

// the default value of concreteVarsOnly should be false
func (d *DsBlockHeader) ToProtobuf(concreteVarsOnly bool) *protobuf.ProtoDSBlockL_DSBlockHeaderL {
	protoDSBlockHeader := &protobuf.ProtoDSBlockL_DSBlockHeaderL{}
	protoBlockHeaderBase := d.BlockHeaderBase.ToProtobuf()
	protoDSBlockHeader.Blockheaderbase = protoBlockHeaderBase

	if !concreteVarsOnly {
		protoDSBlockHeader.Dsdifficulty = d.DsDifficulty
		protoDSBlockHeader.Difficulty = d.Difficulty
		data := make([]byte, 0)
		gasPriceInt, _ := new(big.Int).SetString(d.GasPrice, 10)
		data = UintToByteArray(data, 0, gasPriceInt, 16)
		protoDSBlockHeader.Gasprice = &protobuf.ByteArrayL{
			Data: data,
		}

		var protobufWinners []*protobuf.ProtoDSBlockL_DSBlockHeaderL_PowDSWinnersL
		for key, winner := range d.PoWDSWinners {
			protobufWinner := &protobuf.ProtoDSBlockL_DSBlockHeaderL_PowDSWinnersL{
				Key: &protobuf.ByteArrayL{Data: util.DecodeHex(key)},
				Val: &protobuf.ByteArrayL{Data: winner.Serialize()},
			}
			protobufWinners = append(protobufWinners, protobufWinner)
		}
		protoDSBlockHeader.Dswinners = protobufWinners

		var proposals []*protobuf.ProtoDSBlockL_DSBlockHeaderL_ProposalL
		for proposal, pair := range d.GovDSShardVotesMap {
			protoproposal := &protobuf.ProtoDSBlockL_DSBlockHeaderL_ProposalL{}
			protoproposal.ProposalidL = proposal

			var dsvotes []*protobuf.ProtoDSBlockL_DSBlockHeaderL_VoteL
			for value, count := range pair.First {
				dsvote := &protobuf.ProtoDSBlockL_DSBlockHeaderL_VoteL{
					Value: value,
					Count: count,
				}
				dsvotes = append(dsvotes, dsvote)
			}

			var minerVotes []*protobuf.ProtoDSBlockL_DSBlockHeaderL_VoteL
			for value, count := range pair.Second {
				minerVote := &protobuf.ProtoDSBlockL_DSBlockHeaderL_VoteL{
					Value: value,
					Count: count,
				}
				minerVotes = append(minerVotes, minerVote)
			}

			protoproposal.Dsvotes = dsvotes
			protoproposal.Minervotes = minerVotes
			proposals = append(proposals, protoproposal)
		}

		protoDSBlockHeader.Proposals = proposals

		var dsremoved []*protobuf.ByteArrayL
		for _, key := range d.RemoveDSNodePubKeys {
			dr := &protobuf.ByteArrayL{
				Data: util.DecodeHex(key),
			}
			dsremoved = append(dsremoved, dr)
		}
		protoDSBlockHeader.Dsremoved = dsremoved
	}

	protoDSBlockHeader.Leaderpubkey = &protobuf.ByteArrayL{Data: util.DecodeHex(d.LeaderPubKey)}
	protoDSBlockHeader.Blocknum = d.BlockNum
	protoDSBlockHeader.Epochnum = d.EpochNum

	protoDSBlockHeader.Swinfo = &protobuf.ByteArrayL{Data: d.SwInfo.Serialize()}

	hashset := &protobuf.ProtoDSBlockL_DSBlockHashSetL{
		Shardinghash:  d.DSBlockHashSet.ShardingHash,
		Reservedfield: d.DSBlockHashSet.ReservedField[:],
	}
	protoDSBlockHeader.Hash = hashset

	return protoDSBlockHeader
}
