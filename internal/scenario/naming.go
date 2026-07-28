package scenario

import "fmt"

func numberedNames(prefix string, count int) []string {
	if count < 1 || count > maxRedundantPeers {
		return nil
	}

	names := make([]string, 0, maxRedundantPeers)
	for index := 1; index <= count; index++ {
		names = append(names, numberedName(prefix, index))
	}
	return names
}

func numberedName(prefix string, index int) string {
	return fmt.Sprintf("%s%02d", prefix, index)
}

func accessName(site Site, index int) string {
	return numberedName(site.Code+"-ACC-SW", index)
}

func location(index int) (int, int) {
	return (index-1)/4 + 1, (index-1)%4 + 1
}

func accessPointName(site Site, accessIndex, slot int) string {
	building, floor := location(accessIndex)
	return fmt.Sprintf("%s-WAP-B%02d-F%02d-%02d", site.Code, building, floor, slot)
}

func workstationName(site Site, accessIndex, slot int) string {
	building, floor := location(accessIndex)
	return fmt.Sprintf("%s-WS-B%02d-F%02d-%02d", site.Code, building, floor, slot)
}
