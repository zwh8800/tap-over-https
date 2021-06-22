package main

import "github.com/songgao/water"

var Config = water.Config{
	DeviceType: water.TAP,
	PlatformSpecificParams: water.PlatformSpecificParams{
		Name:   "tap0",
		Driver: water.MacOSDriverTunTapOSX,
	},
}
