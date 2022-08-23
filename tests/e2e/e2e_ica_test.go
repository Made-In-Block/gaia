package e2e

import (
	"cosmossdk.io/math"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"path/filepath"
	"time"
)

// TestICARegister must run before any other
func (s *IntegrationTestSuite) TestICA_1_Register() {
	connectionID := "connection-0"
	var owner string
	s.Run("register_ICA", func() {
		ownerAddr, err := s.chainA.accountsIngenesis[1].keyInfo.GetAddress()
		s.Require().NoError(err)
		owner = ownerAddr.String()
		s.registerICA(owner, connectionID)

		time.Sleep(2 * time.Minute)

		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		s.Require().Eventually(
			func() bool {
				icaAddr, err := queryICAaddr(chainAAPIEndpoint, owner, connectionID)
				s.T().Logf("%s's interchain account on chain %s: %s", owner, s.chainB.id, icaAddr)
				s.Require().NoError(err)
				return owner != "" && icaAddr != ""
			},
			2*time.Minute,
			10*time.Second,
		)
	})
}

func (s *IntegrationTestSuite) TestICA_2_BankSend() {
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))
	connectionID := "connection-0"
	// step 1: get ica addr
	icaOwnerAddr, err := s.chainA.accountsIngenesis[1].keyInfo.GetAddress()
	s.Require().NoError(err)
	icaOnwer := icaOwnerAddr.String()

	var ica string
	s.Require().Eventually(
		func() bool {
			ica, err = queryICAaddr(chainAAPIEndpoint, icaOnwer, connectionID)
			s.Require().NoError(err)

			return err == nil && ica != ""
		},
		time.Minute,
		5*time.Second,
	)

	// step 2: fund ica, send tokens from chain b val to ica on chain b
	senderAddr, err := s.chainB.validators[0].keyInfo.GetAddress()
	s.Require().NoError(err)
	sender := senderAddr.String()

	s.sendMsgSend(s.chainB, 0, sender, ica, tokenAmount.String(), fees.String(), false)

	s.Require().Eventually(
		func() bool {
			afterSenderICAbalance, err := getSpecificBalance(chainBAPIEndpoint, ica, uatomDenom)
			s.Require().NoError(err)
			return afterSenderICAbalance.IsEqual(tokenAmount)
		},
		time.Minute,
		5*time.Second,
	)

	receiver := sender
	var beforeICAsendReceiverbalance sdk.Coin
	s.Require().Eventually(
		func() bool {
			beforeICAsendReceiverbalance, err = getSpecificBalance(chainBAPIEndpoint, receiver, uatomDenom)
			s.Require().NoError(err)

			return !beforeICAsendReceiverbalance.IsNil()
		},
		time.Minute,
		5*time.Second,
	)

	// step 3: prepare ica bank send json
	sendamt := sdk.NewCoin(uatomDenom, math.NewInt(1000000))
	txCmd := []string{
		"gaiad",
		"tx",
		"bank",
		"send",
		ica,
		receiver,
		sendamt.String(),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, ica),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, s.chainA.id),
		"--keyring-backend=test",
	}
	path := filepath.Join(s.chainA.validators[0].configDir(), "config", "ica_bank_send.json")
	s.writeICAtx(txCmd, path)

	// step 4: ica sends some tokens from ica to val on chain b
	s.submitICAtx(icaOnwer, connectionID, "/home/nonroot/.gaia/config/ica_bank_send.json")

	s.Require().Eventually(
		func() bool {
			afterICAsendReceiverbalance, err := getSpecificBalance(chainBAPIEndpoint, receiver, uatomDenom)
			s.Require().NoError(err)

			return afterICAsendReceiverbalance.Sub(beforeICAsendReceiverbalance).IsEqual(sendamt)
		},
		time.Minute,
		5*time.Second,
	)

	// repeat step3: prepare ica ibc send

	// todo add ica delegation after delegation e2e merged

}