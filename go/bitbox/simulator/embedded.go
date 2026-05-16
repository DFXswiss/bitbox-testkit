package simulator

import coresim "github.com/DFXswiss/bitbox-testkit/go/core/simulator"

// embedded is the testkit-curated list of BitBox02 simulator binaries.
// Newest-first. Refresh procedure:
//
//  1. Visit https://github.com/BitBoxSwiss/bitbox02-firmware/releases
//  2. For each new firmware/vX.Y.Z release that ships a simulator asset,
//     download bitbox02-multi-vX.Y.Z-simulator1.0.0-linux-amd64.
//  3. Run `sha256sum` against the file; paste the hex digest into SHA256.
//  4. Prepend the entry to this list (newest at the top).
//
// SHA256s are validated at first-run by core/simulator.Cache.Resolve. A
// mismatched hash produces an explicit error rather than silently
// substituting a tampered binary.
//
// The v9.24.0+ hashes were taken from the BitBox releases page; if a
// download produces a hash-mismatch error, sha256sum the freshly-downloaded
// file and update this list — upstream may have rebuilt the artifact.
var embedded = []coresim.Binary{
	{
		Name:   "bitbox02-multi-v9.26.1-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.26.1/bitbox02-multi-v9.26.1-simulator1.0.0-linux-amd64",
		SHA256: "91ddf47eb0653ce8b3d3344a8e329fc7fef90adfa51e39c5214830cf6e21cccf",
	},
	{
		Name:   "bitbox02-multi-v9.25.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.25.0/bitbox02-multi-v9.25.0-simulator1.0.0-linux-amd64",
		SHA256: "0ba19e82930cc90a8d1f4eb3fc8faf0dc5784ab0a1e4e1b112c90946bd4b3062",
	},
	{
		Name:   "bitbox02-multi-v9.24.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.24.0/bitbox02-multi-v9.24.0-simulator1.0.0-linux-amd64",
		SHA256: "a64cefb90461f479e373b5f5dee4f340f07b4619fbcb55c87784f541b4400e34",
	},
	{
		Name:   "bitbox02-multi-v9.23.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.23.0/bitbox02-multi-v9.23.0-simulator1.0.0-linux-amd64",
		SHA256: "2740eb4be1abd1eb8603478c7a00874f2bff66e620c229348094a427ae8a1fde",
	},
	{
		Name:   "bitbox02-multi-v9.22.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.22.0/bitbox02-multi-v9.22.0-simulator1.0.0-linux-amd64",
		SHA256: "3af12697f6fd51b155bf277ef01ef3eea5290908bff99a4aae83a95cb144ced1",
	},
	{
		Name:   "bitbox02-multi-v9.21.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.21.0/bitbox02-multi-v9.21.0-simulator1.0.0-linux-amd64",
		SHA256: "72031b226ea344970a6a1506893838a63b075e0bad726557ab9d941b42c534f5",
	},
	{
		Name:   "bitbox02-multi-v9.20.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.20.0/bitbox02-multi-v9.20.0-simulator1.0.0-linux-amd64",
		SHA256: "ac32c1a71bd0a3a934bc7b94268f651c655f2e3afbb954811a256e551a420b3d",
	},
	{
		Name:   "bitbox02-multi-v9.19.0-simulator1.0.0-linux-amd64",
		URL:    "https://github.com/BitBoxSwiss/bitbox02-firmware/releases/download/firmware%2Fv9.19.0/bitbox02-multi-v9.19.0-simulator1.0.0-linux-amd64",
		SHA256: "e28be3fd6c7777624ad2574546ba125b7f134f095fa951acc8fb7295f3d33931",
	},
}
