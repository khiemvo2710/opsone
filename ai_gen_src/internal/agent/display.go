package agent

import "opsone/internal/catalog"

func productDisplayName(pc ProductContext) string {
	return catalog.ProductDisplayLabel(pc.Product)
}
