//go:build windows

package main

import (
	_ "embed"
)

//go:embed assets/wallet-rpc-windows.exe
var walletRpcByte []byte
var walletRpcPattern = "tmp*.exe"
