package etherp

const IP_HEADER_LEN = 20
const MAX_IP_PACKET_LEN = 65535

var myMACAddr = func(mac []byte) [8]byte {
	mac = append(mac, 0, 0)
	var data [8]byte
	for i := 0; i < 8; i++ {
		data[i] = mac[i]
	}
	return data
}(myMACSlice)

const (
// 768 = htons(ETH_P_ALL) = htons(3)
// see http://ideone.com/2eunQu

// 17 = AF_PACKET
// see http://ideone.com/TGYlGc


	SOCK_DGRAM      = 2
	SOCK_RAW        = 3
	AF_PACKET       = 17
	HTONS_ETH_P_ALL = 768
	ETHERTYPE_IP    = 0x0800
	ETHERTYPE_APR   = 0x0806
	ETH_ALEN        = 6
)