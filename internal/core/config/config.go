package config

var Protocol = "tcp"
var Port = ":4000"
var MaxConnection = 20000
var MaxKeyNumber = 10000
var EvictionPolicy string = "allkeys-lru"
var EvictionRatio = 0.1
var EpoolMaxSize = 16
var EpoolLruSampleSize = 5
