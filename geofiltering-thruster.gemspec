require_relative "lib/thruster/version"

Gem::Specification.new do |s|
  s.name        = "geofiltering-thruster"
  s.version     = Thruster::VERSION
  s.summary     = "Zero-config HTTP/2 proxy with GeoIP filtering"
  s.description = "A zero-config HTTP/2 proxy for lightweight production deployments with country-based request filtering"
  s.authors     = [ "Kevin McConnell", "Bhal Agashe" ]
  s.email       = "kevin@37signals.com"
  s.homepage    = "https://github.com/bagashe/geofiltering-thruster"
  s.license     = "MIT"

  s.metadata = {
    "homepage_uri" => s.homepage,
    "rubygems_mfa_required" => "true"
  }

  s.platform = "x86_64-linux" # "aarch64-linux"

  s.files = Dir[ "{lib}/**/*", "{exe}/**/*","MIT-LICENSE", "README.md" ]
  s.bindir = "exe"
  s.executables << "thrust"
end
