package com

import (
	"bytes"
	"text/template"
)

type BastionConfig struct {
	OwnerID       string `json:"owner_id"`
	Tag           string `json:"tag"`
	KeyPair       string `json:"keypair"`
	VPNRemote     string `json:"vpn_remote"`
	DNSServer     string `json:"dns_server"`
	NSQDHost      string `json:"nsqd_host"`
	BartnetHost   string `json:"bartnet_host"`
	AuthType      string `json:"auth_type"`
	ModifiedIndex uint64 `json:"modified_index"`
}

const userdata = `#cloud-config
write_files:
  - path: "/etc/opsee/bastion-env.sh"
    permissions: "0644"
    owner: root
    content: |
      CUSTOMER_ID={{.User.CustomerID}}
      CUSTOMER_EMAIL={{.User.Email}}
      BASTION_VERSION={{.Config.Tag}}
      BASTION_ID={{.Bastion.ID}}
      VPN_PASSWORD={{.Bastion.Password}}
      VPN_REMOTE={{.Config.VPNRemote}}
      DNS_SERVER={{.Config.DNSServer}}
      NSQD_HOST={{.Config.NSQDHost}}
      BARTNET_HOST={{.Config.BartnetHost}}
      BASTION_AUTH_TYPE={{.Config.AuthType}}
      GODEBUG=netdns=cgo
{{ with .BastionUsers }}users:{{ range . }}
  - name: {{ .Username }}
    groups:
      - sudo
    ssh-authorized-keys:
      - {{ .Key }}{{ end }}{{ end }}
coreos:
  units:
    - name: "docker.service"
      drop-ins:
        - name: "50-reboot.conf"
          content: |
            [Service]
            FailureAction=reboot-force
  update:
    reboot-strategy: off
    group: beta
`

var userdataTmpl = template.Must(template.New("userdata").Parse(userdata))

func (config *BastionConfig) GenerateUserData(user *User, bastion *Bastion) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})
	var ud = struct {
		User         *User
		BastionUsers []*BastionUser
		Config       *BastionConfig
		Bastion      *Bastion
	}{
		user,
		[]*BastionUser{
			&BastionUser{
				"opsee",
				"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDP+VmyztGmJJTe6YtMtrKazGy3tQC/Pku156Ae10TMzCjvtiol+eL11FKyvNvlENM5EWwIQEng5w3J616kRa92mWr9OWALBn4HJZcztS2YLAXyiC+GLauil6W6xnGzS0DmU5RiYSSPSrmQEwHvmO2umbG190srdaDn/ZvAwptC1br/zc/7ya3XqxHugw1V9kw+KXzTWSC95nPkhOFoaA3nLcMvYWfoTbsU/G08qQy8medqyK80LJJntedpFAYPUrVdGY2J7F2y994YLfapPGzDjM7nR0sRWAZbFgm/BSD0YM8KA0mfGZuKPwKSLMtTUlsmv3l6GJl5a7TkyOlK3zzYtVGO6dnHdZ3X19nldreE3DywpjDrKIfYF2L42FKnpTGFgvunsg9vPdYOiJyIfk6lYsGE6h451OAmV0dxeXhtbqpw4/DsSHtLm5kKjhjRwunuQXEg8SfR3kesJjq6rmhCjLc7bIKm3rSU07zbXSR40JHO1Mc9rqzg2bCk3inJmCKWbMnDvWU1RD475eATEKoG/hv0/7EOywDnFe1m4yi6yZh7XlvakYsxDBPO9/FMlZm2T+cn+TyTmDiw9tEAIEAEiiu18CUNIii1em7XtFDmXjGFWfvteQG/2A98/uDGbmlXd64F2OtU/ulDRJXFGaji8tqxQ/To+2zIeIptLjtqBw==",
			},
		},
		config,
		bastion,
	}

	err := userdataTmpl.Execute(buf, ud)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
