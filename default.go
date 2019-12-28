package flake

import "github.com/liuchong/go-flake/util"

var defaultGen *Generator

func init() {
	ip, err := util.GetIP()
	if err != nil {
		panic(err)
	}

	// A not strictly unique worker Id
	workerId := util.IP4toInt(ip) % (maxWorkerID + 1)

	defaultGen, err = NewGenerator(workerId, 0)
	if err != nil {
		panic(err)
	}
}

func GetDefault() FlakeID {
	return defaultGen.NextID()
}
