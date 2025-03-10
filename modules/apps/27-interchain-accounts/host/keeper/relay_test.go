package keeper_test

import (
	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
)

func (suite *KeeperTestSuite) TestOnRecvPacket() {
	var (
		path       *ibctesting.Path
		packetData []byte
	)

	testCases := []struct {
		msg      string
		malleate func()
		expPass  bool
	}{
		{
			"interchain account successfully executes an arbitrary message type using the * (allow all message types) param",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "Description", sdk.AccAddress(interchainAccountAddr))
				suite.Require().NoError(err)

				suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)

				msg := &govtypes.MsgVote{
					ProposalId: govtypes.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypes.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"*"})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes banktypes.MsgSend",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &banktypes.MsgSend{
					FromAddress: interchainAccountAddr,
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msg := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes stakingtypes.MsgDelegate and stakingtypes.MsgUndelegate sequentially",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				validatorAddr := (sdk.ValAddress)(suite.chainB.Vals.Validators[0].Address)
				msgDelegate := &stakingtypes.MsgDelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				msgUndelegate := &stakingtypes.MsgUndelegate{
					DelegatorAddress: interchainAccountAddr,
					ValidatorAddress: validatorAddr.String(),
					Amount:           sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000)),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msgDelegate, msgUndelegate}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msgDelegate), sdk.MsgTypeURL(msgUndelegate)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgSubmitProposal",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				protoAny, err := codectypes.NewAnyWithValue(testProposal)
				suite.Require().NoError(err)

				msg := &govtypes.MsgSubmitProposal{
					Content:        protoAny,
					InitialDeposit: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Proposer:       interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes govtypes.MsgVote",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				// Populate the gov keeper in advance with an active proposal
				testProposal := &govtypes.TextProposal{
					Title:       "IBC Gov Proposal",
					Description: "tokens for all!",
				}

				proposalMsg, err := govv1.NewLegacyContent(testProposal, interchainAccountAddr)
				suite.Require().NoError(err)

				proposal, err := govv1.NewProposal([]sdk.Msg{proposalMsg}, govtypes.DefaultStartingProposalID, suite.chainA.GetContext().BlockTime(), suite.chainA.GetContext().BlockTime(), "test proposal", "title", "description", sdk.AccAddress(interchainAccountAddr))
				suite.Require().NoError(err)

				suite.chainB.GetSimApp().GovKeeper.SetProposal(suite.chainB.GetContext(), proposal)
				suite.chainB.GetSimApp().GovKeeper.ActivateVotingPeriod(suite.chainB.GetContext(), proposal)

				msg := &govtypes.MsgVote{
					ProposalId: govtypes.DefaultStartingProposalID,
					Voter:      interchainAccountAddr,
					Option:     govtypes.OptionYes,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgFundCommunityPool",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgFundCommunityPool{
					Amount:    sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(5000))),
					Depositor: interchainAccountAddr,
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes disttypes.MsgSetWithdrawAddress",
			func() {
				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &disttypes.MsgSetWithdrawAddress{
					DelegatorAddress: interchainAccountAddr,
					WithdrawAddress:  suite.chainB.SenderAccount.GetAddress().String(),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"interchain account successfully executes transfertypes.MsgTransfer",
			func() {
				transferPath := ibctesting.NewPath(suite.chainB, suite.chainC)
				transferPath.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
				transferPath.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
				transferPath.EndpointA.ChannelConfig.Version = transfertypes.Version
				transferPath.EndpointB.ChannelConfig.Version = transfertypes.Version

				suite.coordinator.Setup(transferPath)

				interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, path.EndpointA.ChannelConfig.PortID)
				suite.Require().True(found)

				msg := &transfertypes.MsgTransfer{
					SourcePort:       transferPath.EndpointA.ChannelConfig.PortID,
					SourceChannel:    transferPath.EndpointA.ChannelID,
					Token:            sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)),
					Sender:           interchainAccountAddr,
					Receiver:         suite.chainA.SenderAccount.GetAddress().String(),
					TimeoutHeight:    clienttypes.NewHeight(1, 100),
					TimeoutTimestamp: uint64(0),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			true,
		},
		{
			"unregistered sdk.Msg",
			func() {
				msg := &banktypes.MsgSendResponse{}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{"/" + proto.MessageName(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			false,
		},
		{
			"cannot unmarshal interchain account packet data",
			func() {
				packetData = []byte{}
			},
			false,
		},
		{
			"cannot deserialize interchain account packet data messages",
			func() {
				data := []byte("invalid packet data")

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"invalid packet type - UNSPECIFIED",
			func() {
				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.UNSPECIFIED,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: interchain account not found for controller port ID",
			func() {
				path.EndpointA.ChannelConfig.PortID = "invalid-port-id"

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{&banktypes.MsgSend{}}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: message type not allowed", // NOTE: do not update params to explicitly force the error
			func() {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()
			},
			false,
		},
		{
			"unauthorised: signer address is not the interchain account associated with the controller portID",
			func() {
				msg := &banktypes.MsgSend{
					FromAddress: suite.chainB.SenderAccount.GetAddress().String(), // unexpected signer
					ToAddress:   suite.chainB.SenderAccount.GetAddress().String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))),
				}

				data, err := icatypes.SerializeCosmosTx(suite.chainA.GetSimApp().AppCodec(), []proto.Message{msg}, icatypes.EncodingProtobuf)
				suite.Require().NoError(err)

				icaPacketData := icatypes.InterchainAccountPacketData{
					Type: icatypes.EXECUTE_TX,
					Data: data,
				}

				packetData = icaPacketData.GetBytes()

				params := types.NewParams(true, []string{sdk.MsgTypeURL(msg)})
				suite.chainB.GetSimApp().ICAHostKeeper.SetParams(suite.chainB.GetContext(), params)
			},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		suite.Run(tc.msg, func() {
			suite.SetupTest() // reset

			path = NewICAPath(suite.chainA, suite.chainB)
			suite.coordinator.SetupConnections(path)

			err := SetupICAPath(path, TestOwnerAddress)
			suite.Require().NoError(err)

			portID, err := icatypes.NewControllerPortID(TestOwnerAddress)
			suite.Require().NoError(err)

			// Get the address of the interchain account stored in state during handshake step
			storedAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(suite.chainB.GetContext(), ibctesting.FirstConnectionID, portID)
			suite.Require().True(found)

			icaAddr, err := sdk.AccAddressFromBech32(storedAddr)
			suite.Require().NoError(err)

			// Check if account is created
			interchainAccount := suite.chainB.GetSimApp().AccountKeeper.GetAccount(suite.chainB.GetContext(), icaAddr)
			suite.Require().Equal(interchainAccount.GetAddress().String(), storedAddr)

			suite.fundICAWallet(suite.chainB.GetContext(), path.EndpointA.ChannelConfig.PortID, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(10000))))

			tc.malleate() // malleate mutates test data

			packet := channeltypes.NewPacket(
				packetData,
				suite.chainA.SenderAccount.GetSequence(),
				path.EndpointA.ChannelConfig.PortID,
				path.EndpointA.ChannelID,
				path.EndpointB.ChannelConfig.PortID,
				path.EndpointB.ChannelID,
				clienttypes.NewHeight(1, 100),
				0,
			)

			txResponse, err := suite.chainB.GetSimApp().ICAHostKeeper.OnRecvPacket(suite.chainB.GetContext(), packet)

			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(txResponse)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(txResponse)
			}
		})
	}
}

func (suite *KeeperTestSuite) fundICAWallet(ctx sdk.Context, portID string, amount sdk.Coins) {
	interchainAccountAddr, found := suite.chainB.GetSimApp().ICAHostKeeper.GetInterchainAccountAddress(ctx, ibctesting.FirstConnectionID, portID)
	suite.Require().True(found)

	msgBankSend := &banktypes.MsgSend{
		FromAddress: suite.chainB.SenderAccount.GetAddress().String(),
		ToAddress:   interchainAccountAddr,
		Amount:      amount,
	}

	res, err := suite.chainB.SendMsgs(msgBankSend)
	suite.Require().NotEmpty(res)
	suite.Require().NoError(err)
}
