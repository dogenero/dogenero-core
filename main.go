package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/gabstv/httpdigest"
	"github.com/gotk3/gotk3/gtk"
	"gitlab.com/moneropay/go-monero/walletrpc"
)

const coinTicker string = "dogenero"
const RpcPort string = "27883"
const daemonAddress string = "23.137.250.135:18881"

var stack *gtk.Stack
var loadWalletGrid *gtk.Grid
var createWalletGrid *gtk.Grid

var client *walletrpc.Client

var rpc_username string
var rpc_password string

var cmd *exec.Cmd

var win *gtk.Window

var tempFilePath string

func main() {
	f, err := os.CreateTemp("", walletRpcPattern) // in Go version older than 1.17 you can use ioutil.TempFile
	if err != nil {
		log.Fatal(err)
	}

	f.Write(walletRpcByte)
	tempFilePath = f.Name()
	f.Close()
	walletRpcByte = nil
	os.Chmod(tempFilePath, 0777)

	rpc_username = GenerateRandomString(8)
	rpc_password = GenerateRandomString(14)
	go runWallet()
	runRpc()

	gtk.Init(nil)

	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetBorderWidth(10)
	win.SetTitle("Dogenero Core")
	win.Connect("destroy", func() {
		cmd.Process.Kill()
		os.Remove(tempFilePath)
		gtk.MainQuit()

	})
	time.Sleep(500 * time.Millisecond)
	os.Remove(tempFilePath)

	stack, _ = gtk.StackNew()
	win.Add(stack)

	grid, _ := gtk.GridNew()
	stack.AddNamed(grid, "page 0")
	firstWindow(grid)

	mainWalletGrid, _ := gtk.GridNew()
	stack.AddNamed(mainWalletGrid, "mainWallet")
	mainWalletWindow(mainWalletGrid)

	loadWalletGrid, _ = gtk.GridNew()
	stack.AddNamed(loadWalletGrid, "loadWallet")
	loadWalletWindow()

	createWalletGrid, _ = gtk.GridNew()
	stack.AddNamed(createWalletGrid, "createWallet")
	createWalletWindow()

	win.SetDefaultSize(800, 600)
	win.ShowAll()
	gtk.Main()
}

func formatMnemonic(mnemonic string) string {
	var output string
	var spacesSoFar int = 0
	var breaks int = 0

	for _, d := range mnemonic {
		if d == ' ' {
			spacesSoFar += 1
		}
		if spacesSoFar >= 8 && breaks < 2 {
			output = output + "\n"
			spacesSoFar = 0
			breaks += 1
		} else {
			output = output + string(d)
		}
	}
	return output
}

func runWallet() {
	cmd = exec.Command(tempFilePath, "--rpc-bind-port", RpcPort, "--wallet-dir",
		"./wallets", "--rpc-login", rpc_username+":"+rpc_password, "--log-file", "./wallets/rpc.log",
		"--daemon-address", daemonAddress)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

}
func runRpc() {
	client = walletrpc.New(walletrpc.Config{
		Address: "http://127.0.0.1:" + RpcPort + "/json_rpc",
		Client: &http.Client{
			Transport: httpdigest.New(rpc_username, rpc_password),
		},
	})
	go setDaemon()
}
func setDaemon() {
	client.SetDaemon(context.Background(), &walletrpc.SetDaemonRequest{
		Address: daemonAddress,
		Trusted: false,
	})
}

func firstWindow(grid *gtk.Grid) {
	titleLabel, _ := gtk.LabelNew("")
	titleLabel.SetMarkup("<span size=\"x-large\">Dogenero Core</span>")
	grid.Attach(titleLabel, 0, 0, 1, 1)

	emptyLabel, _ := gtk.LabelNew("")
	grid.Attach(emptyLabel, 0, 1, 1, 1)

	loadWalletBtn := setup_btn("Open wallet", func() {
		stack.SetVisibleChild(loadWalletGrid)
	})
	loadWalletBtn.SetMarginTop(10)
	grid.Attach(loadWalletBtn, 0, 2, 1, 1)

	emptyLabel2, _ := gtk.LabelNew("")
	grid.Attach(emptyLabel2, 0, 3, 1, 1)

	newWalletBtn := setup_btn("Create new wallet", func() {
		stack.SetVisibleChild(createWalletGrid)
	})
	grid.Attach(newWalletBtn, 0, 4, 1, 1)

	grid.SetHAlign(gtk.ALIGN_CENTER)
	grid.SetVAlign(gtk.ALIGN_CENTER)

}

var addressEntry *gtk.Entry

func initWallet() {
	setDaemon()
	addressRes, err := client.GetAddress(context.Background(), &walletrpc.GetAddressRequest{AccountIndex: 0})
	if err != nil {
		panic(err)
	}
	addressEntry.SetText(addressRes.Address)
	go balanceRefresher()
	stack.SetVisibleChildName("mainWallet")
}

var balanceLabel *gtk.Label

func mainWalletWindow(grid *gtk.Grid) {

	balanceLabel, _ = gtk.LabelNew("Balance: ? " + coinTicker)
	grid.Attach(balanceLabel, 0, 0, 1, 1)

	receiveTabGrid, _ := gtk.GridNew()

	addressEntry, _ = gtk.EntryNew()
	addressEntry.SetEditable(false)
	addressEntry.SetCanFocus(false)
	addressEntry.SetHExpand(true)
	addressEntry.SetWidthChars(30)
	receiveTabGrid.Attach(addressEntry, 0, 0, 1, 1)

	sendTabGrid, _ := gtk.GridNew()

	emptyL := setup_label("")
	emptyL.SetMarginStart(10)
	sendTabGrid.Attach(emptyL, 0, 0, 1, 1)

	sendTabGrid.Attach(setupMarginLabel("Address  "), 0, 1, 1, 1)
	sendAddressEntry, _ := gtk.EntryNew()
	sendAddressEntry.SetHExpand(true)
	sendAddressEntry.SetMarginEnd(15)
	sendTabGrid.Attach(sendAddressEntry, 1, 1, 5, 1)

	sendTabGrid.Attach(setup_label(""), 0, 2, 1, 1)

	sendTabGrid.Attach(setupMarginLabel("Amount "), 0, 3, 1, 1)
	sendAmountEntry, _ := gtk.EntryNew()
	sendTabGrid.Attach(sendAmountEntry, 1, 3, 1, 1)

	sendTabGrid.Attach(setup_label(""), 0, 4, 1, 1)

	settingsTabGrid, _ := gtk.GridNew()
	settingsTabGrid.Attach(setup_btn("View seed phrase", func() {
		dialog := create_popup(gtk.MESSAGE_QUESTION, "Please enter your password.")
		contArea, _ := dialog.GetContentArea()

		passwdEntry, _ := gtk.EntryNew()
		passwdEntry.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
		passwdEntry.SetVisibility(false)
		passwdEntry.SetMarginStart(10)
		passwdEntry.SetMarginEnd(10)
		contArea.PackStart(passwdEntry, true, true, 5)
		passwdEntry.Show()

		dialog.Show()
		dialog.Run()
		passwd, _ := passwdEntry.GetText()
		if passwd != walletPassword {
			show_popup(gtk.MESSAGE_ERROR, "Incorrect password")
			return
		}
		dialog.Destroy()
		mnemonic, er := client.QueryKey(context.Background(), &walletrpc.QueryKeyRequest{
			KeyType: "mnemonic",
		})
		if er != nil {
			show_popup(gtk.MESSAGE_ERROR, er.Error())
			return
		}
		mnemonicDialog := create_popup(gtk.MESSAGE_OTHER, "Mnemonic phrase")
		mnemonicDialog.SetDefaultSize(500, 200)
		mArea, _ := mnemonicDialog.GetContentArea()

		mTextBox, _ := gtk.TextViewNew()
		mTextBox.SetMarginStart(10)
		mTextBox.SetMarginEnd(10)
		mTextBuffer, _ := mTextBox.GetBuffer()
		mTextBuffer.SetText(formatMnemonic(mnemonic.Key))
		mTextBox.SetEditable(false)
		mArea.PackStart(mTextBox, true, true, 5)
		mTextBox.Show()
		mnemonicDialog.Show()
		mnemonicDialog.Run()
		mnemonicDialog.Destroy()
	}), 0, 0, 1, 1)

	sendButton := setup_btn("Send", func() {
		amtp, _ := sendAmountEntry.GetText()
		amtFloat, err := strconv.ParseFloat(amtp, 64)
		if err != nil {
			show_popup(gtk.MESSAGE_ERROR, "Cannot parse amount.")
		}
		amt := uint64(amtFloat * math.Pow(10, 12))
		addr, _ := sendAddressEntry.GetText()

		data, err2 := client.Transfer(context.Background(), &walletrpc.TransferRequest{
			Destinations:  []walletrpc.Destination{{Amount: amt, Address: addr}},
			DoNotRelay:    true,
			GetTxMetadata: true,
		})
		if err2 != nil {
			show_popup(gtk.MESSAGE_ERROR, err2.Error())
			return
		}

		dialog := gtk.MessageDialogNew(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_QUESTION, gtk.BUTTONS_OK_CANCEL,
			"Are you sure you want to send "+formatAmount(data.Amount)+" "+coinTicker+" (fee "+formatAmount(data.Fee)+") ?")
		dialog.Show()
		resType := dialog.Run()
		dialog.Destroy()
		if resType == gtk.RESPONSE_OK {
			fmt.Println("Sending " + strconv.FormatUint(data.Amount/10^12, 10) + " (fee " + formatAmount(data.Fee) + ") to " + addr)
			relayTxData, relayErr := client.RelayTx(context.Background(), &walletrpc.RelayTxRequest{Hex: data.TxMetadata})
			if relayErr != nil {
				show_popup(gtk.MESSAGE_ERROR, relayErr.Error())
				return
			} else {
				fmt.Println(relayTxData)
				show_popup(gtk.MESSAGE_INFO, "Sent transaction. txid: "+relayTxData.TxHash)
			}
		} else {
			fmt.Println("Transaction cancelled.")
		}
	})
	sendButton.SetMarginBottom(10)
	sendTabGrid.Attach(sendButton, 1, 5, 1, 1)

	tabSwitcher, _ := gtk.NotebookNew()
	tabSwitcher.SetHExpand(true)
	tabSwitcher.SetBorderWidth(10)
	receiveTabLabel, _ := gtk.LabelNew("Receive")
	sendTabLabel, _ := gtk.LabelNew("Send")
	settingsTabLabel, _ := gtk.LabelNew("Settings")
	tabSwitcher.AppendPage(receiveTabGrid, receiveTabLabel)
	tabSwitcher.AppendPage(sendTabGrid, sendTabLabel)
	tabSwitcher.AppendPage(settingsTabGrid, settingsTabLabel)

	//win_width, win_height := win.GetSize()

	grid.Attach(tabSwitcher, 0, 1, 1, 1)
}
func formatAmount(a uint64) string {
	return strconv.FormatFloat(float64(a)/math.Pow(10, 12), 'f', 4, 64)
}
func balanceRefresher() {
	refreshBalance := func() {
		d, e := client.GetBalance(context.Background(), &walletrpc.GetBalanceRequest{AccountIndex: 0})
		if e != nil {
			fmt.Println(e)
			return
		}
		if d.Balance > d.UnlockedBalance {
			balanceLabel.SetText("Balance: " + formatAmount(d.UnlockedBalance) + " " + coinTicker + " (+" + formatAmount(d.Balance-d.UnlockedBalance) + " unconfirmed)")
		} else {
			balanceLabel.SetText("Balance: " + formatAmount(d.Balance) + " " + coinTicker)
		}
	}

	for {
		if stack.GetVisibleChildName() != "mainWallet" {
			return
		}
		refreshBalance()
		time.Sleep(5 * time.Second)
	}
}

func loadWalletWindow() {
	loadWalletGrid.SetHAlign(gtk.ALIGN_CENTER)
	loadWalletGrid.SetVAlign(gtk.ALIGN_CENTER)

	loadWalletNameEntry, _ := gtk.EntryNew()
	loadWalletNameEntry.SetText("wallet")
	loadWalletGrid.Attach(loadWalletNameEntry, 1, 0, 2, 1)

	loadWalletPasswordEntry, _ := gtk.EntryNew()
	loadWalletGrid.Attach(loadWalletPasswordEntry, 1, 2, 2, 1)
	loadWalletPasswordEntry.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	loadWalletPasswordEntry.SetVisibility(false)

	loadWalletNameLabel := setup_label("Wallet file name  ")
	loadWalletGrid.Attach(loadWalletNameLabel, 0, 0, 1, 1)

	loadWalletGrid.Attach(add_spacing(), 1, 1, 1, 1)

	loadWalletPasswordLabel := setup_label("Wallet password  ")
	loadWalletGrid.Attach(loadWalletPasswordLabel, 0, 2, 1, 1)

	loadWalletGrid.Attach(add_spacing(), 1, 3, 1, 1)

	loadWalletButton := setup_btn("Load wallet", func() {
		walletName, _ := loadWalletNameEntry.GetText()
		walletPass, _ := loadWalletPasswordEntry.GetText()
		loadWallet(walletName, walletPass)
	})
	loadWalletGrid.Attach(loadWalletButton, 1, 4, 1, 1)

	backBtn := setup_btn("Back", func() {
		stack.SetVisibleChildName("page 0")
	})
	backBtn.SetMarginStart(10)
	loadWalletGrid.Attach(backBtn, 2, 4, 1, 1)

}

var walletPassword string

func loadWallet(walletName string, walletPasswd string) {
	err := client.OpenWallet(context.Background(), &walletrpc.OpenWalletRequest{Filename: walletName, Password: walletPasswd})
	if err != nil {
		dialog := gtk.MessageDialogNew(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, err.Error())
		fmt.Println(err)
		dialog.Show()
		dialog.Run()
		dialog.Destroy()
	} else {
		walletPassword = walletPasswd
		initWallet()

	}
}

func createWalletWindow() {
	createWalletGrid.SetHAlign(gtk.ALIGN_CENTER)
	createWalletGrid.SetVAlign(gtk.ALIGN_CENTER)

	createWalletNameEntry, _ := gtk.EntryNew()
	createWalletNameEntry.SetText("wallet")
	createWalletGrid.Attach(createWalletNameEntry, 1, 0, 2, 1)

	createWalletPasswordEntry, _ := gtk.EntryNew()
	createWalletGrid.Attach(createWalletPasswordEntry, 1, 2, 2, 1)
	createWalletPasswordEntry.SetInputPurpose(gtk.INPUT_PURPOSE_PASSWORD)
	createWalletPasswordEntry.SetVisibility(false)

	createWalletNameLabel := setup_label("Wallet file name  ")
	createWalletGrid.Attach(createWalletNameLabel, 0, 0, 1, 1)

	createWalletGrid.Attach(add_spacing(), 1, 1, 1, 1)

	createWalletPasswordLabel := setup_label("Wallet password  ")
	createWalletGrid.Attach(createWalletPasswordLabel, 0, 2, 1, 1)

	createWalletGrid.Attach(add_spacing(), 1, 3, 1, 1)

	createWalletButton := setup_btn("Create wallet", func() {
		os.Mkdir("./wallets", 0700)
		walletName, _ := createWalletNameEntry.GetText()
		walletPass, _ := createWalletPasswordEntry.GetText()
		createWallet(walletName, walletPass)
	})
	createWalletGrid.Attach(createWalletButton, 1, 4, 1, 1)

	backBtn := setup_btn("Back", func() {
		stack.SetVisibleChildName("page 0")
	})
	backBtn.SetMarginStart(10)
	createWalletGrid.Attach(backBtn, 2, 4, 1, 1)

}

func createWallet(walletName string, newWalletPassword string) {
	err := client.CreateWallet(context.Background(), &walletrpc.CreateWalletRequest{Filename: walletName, Password: newWalletPassword, Language: "English"})
	if err != nil {
		dialog := gtk.MessageDialogNew(win, gtk.DIALOG_DESTROY_WITH_PARENT, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, err.Error())
		fmt.Println(err)
		dialog.Show()
		dialog.Run()
		dialog.Destroy()
	} else {
		initWallet()
	}
}

func setup_btn(label string, onClick func()) *gtk.Button {
	btn, err := gtk.ButtonNewWithLabel(label)
	if err != nil {
		log.Fatal("Unable to create button: ", err)
	}
	btn.Connect("clicked", onClick)
	return btn
}
func setup_label(text string) *gtk.Label {
	label, err := gtk.LabelNew(text)
	if err != nil {
		log.Fatal("Unable to create label: ", err)
	}
	return label
}
func setupMarginLabel(text string) *gtk.Label {
	label, err := gtk.LabelNew(text)
	if err != nil {
		log.Fatal("Unable to create label: ", err)
	}
	label.SetMarginStart(5)
	return label
}
func add_spacing() *gtk.Label {
	label, err := gtk.LabelNew(" ")
	if err != nil {
		log.Fatal("Unable to create label: ", err)
	}
	return label
}
func show_popup(msgtype gtk.MessageType, text string) {
	dialog := gtk.MessageDialogNew(win, gtk.DIALOG_DESTROY_WITH_PARENT, msgtype, gtk.BUTTONS_OK, text)
	dialog.Show()
	dialog.Run()
	dialog.Destroy()
}

func create_popup(msgtype gtk.MessageType, text string) *gtk.MessageDialog {
	dialog := gtk.MessageDialogNew(win, gtk.DIALOG_DESTROY_WITH_PARENT, msgtype, gtk.BUTTONS_OK, text)
	return dialog
}
