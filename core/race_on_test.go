//go:build race

package core

// raceEnabled reports whether the binary was built with -race. sing-box has a
// data race of its own in route.NetworkManager.Start (sing-box@v1.14.0
// route/network.go:220 vs :574, where Start writes state its own interface
// monitor callback is already reading), so any test that starts a real Box
// reports a race that is not this repo's. Those tests skip under -race and run
// normally otherwise; TestCoreLockConcurrency covers Core's own locking in a
// way that does not need a started box, so it runs in both modes.
const raceEnabled = true
