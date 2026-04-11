package config

const NILL = "$-1\r\n"

var Protocol = "tcp"
var Port = ":4000"
var MaxConnection = 20000
var MaxKeyNumber = 10000
var EvictionPolicy string = "allkeys-lru"
var EvictionRatio = 0.1
var EpoolMaxSize = 16
var EpoolLruSampleSize = 5

const ServerStatusIdle = 1
const ServerStatusBusy = 2
const ServerStatusShuttingDown = 3
