{
	"cdiVersion": "0.5.0",
	"kind": "apptainertesting.sylabs.io/device",

	"devices": [
		{
			"name": "TesterDevice",
			"containerEdits": {
				"deviceNodes": {{tojson .DeviceNodes}},
				"mounts": {{tojson .Mounts}}
			}
		}
	],

	"containerEdits": {
		"env": {{tojson .Env}}
	}
}
