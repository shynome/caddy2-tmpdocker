package tmpdocker

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("tmpdocker", parseCaddyfile)
}

// parseCaddyfile parses the tmpd directive.
//
//    tmpdocker [<service_name>] {
//        name         <service_name>
//	      timeout      <freeze_timeout>
//	      docker_host  <DockerHost>
//    }
//
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var tmpd TmpDocker

	for h.Next() {
		args := h.RemainingArgs()
		switch len(args) {
		case 0:
		case 1:
			tmpd.ServiceName = args[0]
		default:
			return nil, h.ArgErr()
		}

		for h.NextBlock(0) {
			switch h.Val() {
			case "name":
				if !h.Args(&tmpd.ServiceName) {
					return nil, h.ArgErr()
				}
			case "timeout":
				{
					var s string
					if !h.Args(&s) {
						return nil, h.ArgErr()
					}
					var timeout caddy.Duration
					timeout.UnmarshalJSON([]byte(s))
					tmpd.FreezeTimeout = timeout
				}
			case "docker_host":
				if !h.Args(&tmpd.DockerHost) {
					return nil, h.ArgErr()
				}
			default:
				return nil, h.Errf("unknown subdirective '%s'", h.Val())
			}
		}
	}

	return &tmpd, nil
}
