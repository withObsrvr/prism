package pagesv2

import "net/url"

// txFragmentHref builds the URL htmx uses to load a section of the transaction
// page.
//
// The network has to travel in the URL. A fragment request is a fresh HTTP
// call that inherits nothing from the page that issued it, so without the
// parameter it resolves the network from the cookie and falls back to the
// default — which means a page opened at ?network=testnet loads its shell from
// testnet and then asks mainnet for the contents. The transaction is absent
// there, every fragment fails, and the page sits on its loading skeleton
// forever with no indication of why.
//
// Carrying it explicitly also makes the URL shareable: the recipient sees the
// same network as the sender, whatever their cookie says.
func txFragmentHref(hash, section, network string) string {
	href := "/fragments/tx/v2/" + url.PathEscape(hash) + "/" + section
	if network == "" {
		return href
	}
	return href + "?network=" + url.QueryEscape(network)
}
