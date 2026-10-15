package scenario

import "github.com/MustardSeedNetworks/niac-go/internal/converter"

// packFTPBannerJetDirect is the greeting HP Jetdirect sends on FTP printing.
// It is documented, not captured: HP Jetdirect Print Servers Administrator's
// Guide (ENWW, Edition 3, 2004), "FTP Printing", "Example of an FTP Session",
// p. 181, https://research.iat.sfu.ca/research/images/c/cf/HPadmin.pdf. The
// lab's FutureSmart printer ships FTP off, so it had none to capture.
const packFTPBannerJetDirect = "220 JD FTP Server Ready"

// packFTP opens FTP printing on the HP printers. No other pack printer or
// server gets FTP: Zebra, Epson, Synology and vsftpd publish no greeting, and
// a greeting is never invented.
func packFTP(profile DeviceProfile) *converter.FtpConfig {
	if profile.DeviceType != "printer" || profile.Vendor != "hewlett packard" {
		return nil
	}

	return &converter.FtpConfig{Enabled: true, WelcomeBanner: packFTPBannerJetDirect}
}
