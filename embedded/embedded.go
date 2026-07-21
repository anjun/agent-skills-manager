package embedded

import _ "embed"

//go:embed data/vendor-skills.yaml
var VendorSkillsYAML []byte
