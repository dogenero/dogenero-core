//go:build linux

package main

import (
	_ "embed"
)

//go:embed assets/wallet-rpc-linux
var walletRpcByte []byte
var walletRpcPattern = "tmp"
