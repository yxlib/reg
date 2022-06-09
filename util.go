package reg

import (
	"fmt"
	"strconv"
	"strings"
)

func GetSrvTypeKey(srvType uint32) string {
	return fmt.Sprintf("/%d", srvType)
}

func GetSrvKey(srvType uint32, srvNo uint32) string {
	return fmt.Sprintf("/%d/%d", srvType, srvNo)
}

func GetSrvTypeAndNo(key string) (uint32, uint32) {
	subPaths := ParseInfoPath(key)
	if len(subPaths) < 2 {
		return 0, 0
	}

	peerType, err := strconv.ParseUint(subPaths[0], 10, 32)
	if err != nil {
		return 0, 0
	}

	peerNo, err := strconv.ParseUint(subPaths[1], 10, 32)
	if err != nil {
		return 0, 0
	}

	return uint32(peerType), uint32(peerNo)
}

func ParseInfoPath(path string) []string {
	subPaths := make([]string, 0)

	if len(path) <= 1 {
		return subPaths
	}

	idx := strings.Index(path, "/")
	if idx != 0 {
		return subPaths
	}

	// subPath = append(subPath, "/")
	// if len(path) == 1 {
	// 	return subPath
	// }

	subStr := strings.Split(path[1:], "/")
	return append(subPaths, subStr...)
}
