package simulator

import coresim "github.com/joshuakrueger-dfx/bitbox-testkit/go/core/simulator"

// embedded mirrors upstream bitbox02-api-go/api/firmware/testdata/simulators.json
// at the pinned version. Refresh by re-running scripts/update-simulators.sh
// (TODO) and re-pasting; the SHA256 of every binary is verified at launch.
var embedded = []coresim.Binary{
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
