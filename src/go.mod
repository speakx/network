module network

go 1.13

replace environment => ../../environment/src

replace idgenerator => ../../idgenerator/src

replace mmapcache => ../../mmapcache/src

replace single => ../../single/src

replace singledb => ../../singledb/src

replace svrdemo => ../../svrdemo/src

replace github.com/panjf2000/gnet => ../../pkg/github.com/panjf2000/gnet

require (
	environment v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.0.0-20191210023423-ac6580df4449
)
