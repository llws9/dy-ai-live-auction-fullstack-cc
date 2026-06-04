package service

import "time"

var auctionBusinessLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

func auctionBusinessNow() time.Time {
	return time.Now().In(auctionBusinessLocation)
}
