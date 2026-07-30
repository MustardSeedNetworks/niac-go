package walkcapture

import (
	"fmt"
	"strings"

	"github.com/gosnmp/gosnmp"
)

const (
	maxUsername = 32
	minPassword = 8
)

func configureV3(client *gosnmp.GoSNMP, request Request) error {
	if request.Username == "" || len(request.Username) > maxUsername {
		return fmt.Errorf("%w: SNMPv3 username is required", ErrInvalidRequest)
	}
	auth, err := authProtocol(request.AuthProtocol)
	if err != nil {
		return err
	}
	privacy, err := privProtocol(request.PrivProtocol)
	if err != nil {
		return err
	}
	flags := gosnmp.NoAuthNoPriv
	if auth != gosnmp.NoAuth {
		if len(request.AuthPassword) < minPassword {
			return fmt.Errorf(
				"%w: authentication password must contain at least 8 characters",
				ErrInvalidRequest,
			)
		}
		flags = gosnmp.AuthNoPriv
	}
	if privacy != gosnmp.NoPriv {
		if auth == gosnmp.NoAuth || len(request.PrivPassword) < minPassword {
			return fmt.Errorf(
				"%w: privacy requires authentication and an 8-character password",
				ErrInvalidRequest,
			)
		}
		flags = gosnmp.AuthPriv
	}
	client.Version = gosnmp.Version3
	client.MsgFlags = flags
	client.SecurityModel = gosnmp.UserSecurityModel
	client.SecurityParameters = &gosnmp.UsmSecurityParameters{
		UserName: request.Username, AuthenticationProtocol: auth,
		AuthenticationPassphrase: request.AuthPassword, PrivacyProtocol: privacy,
		PrivacyPassphrase: request.PrivPassword,
	}
	return nil
}

func authProtocol(value string) (gosnmp.SnmpV3AuthProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return gosnmp.NoAuth, nil
	case "md5":
		return gosnmp.MD5, nil
	case "sha", "sha1":
		return gosnmp.SHA, nil
	case "sha224":
		return gosnmp.SHA224, nil
	case "sha256":
		return gosnmp.SHA256, nil
	case "sha384":
		return gosnmp.SHA384, nil
	case "sha512":
		return gosnmp.SHA512, nil
	default:
		return gosnmp.NoAuth, fmt.Errorf(
			"%w: unsupported authentication protocol",
			ErrInvalidRequest,
		)
	}
}

func privProtocol(value string) (gosnmp.SnmpV3PrivProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return gosnmp.NoPriv, nil
	case "des":
		return gosnmp.DES, nil
	case "aes", "aes128":
		return gosnmp.AES, nil
	case "aes192":
		return gosnmp.AES192, nil
	case "aes256":
		return gosnmp.AES256, nil
	default:
		return gosnmp.NoPriv, fmt.Errorf("%w: unsupported privacy protocol", ErrInvalidRequest)
	}
}
