const fs = require('fs');
const path = require('path');

function indexFolder(folder) {
	const result = [];
	try {
		for (const item of fs.readdirSync(folder, { withFileTypes: true })) {
			if (item.isDirectory() && item.name[0] !== '.') {
				result.push({
					name: item.name + '@v?',
					children: indexFolder(path.join(folder, item.name, 'node_modules')),
				});
			}
		}
	} catch (e) {
		if (e.code !== 'ENOENT') throw e;
	}

	return result;
}

console.log(JSON.stringify({
	type: 'tree',
	data: {
		trees: indexFolder('node_modules'),
	},
}));
