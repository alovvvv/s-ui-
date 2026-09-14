package anytls

import (
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"

	anytls "github.com/anytls/sing-anytls"
)

func (h *Inbound) UpdateUsers(users []option.AnyTLSUser) error {
	h.service.UpdateUsers(common.Map(users, func(it option.AnyTLSUser) anytls.User {
		return (anytls.User)(it)
	}))
	return nil
}
