module gatewayauth

go 1.22

require (
	confload v0.0.0
	tracelog v0.0.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace confload => ../confload

replace tracelog => ../tracelog
