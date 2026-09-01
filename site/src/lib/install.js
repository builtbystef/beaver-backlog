import { copyFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(here, '../../..');
const scriptNames = ['install.sh', 'install.ps1'];

export function unixInstallCommand(origin) {
	return `curl -fsSL ${new URL('/install.sh', origin).href} | sh`;
}

export function windowsInstallCommand(origin) {
	return `irm ${new URL('/install.ps1', origin).href} | iex`;
}

export function installScriptsIntegration() {
	return {
		name: 'install-scripts',
		hooks: {
			// Copy from the repository root so site/ never holds a second
			// committed copy that can drift from the scripts CI lints.
			'astro:build:done': ({ dir }) => {
				const dest = fileURLToPath(dir);
				for (const name of scriptNames) {
					copyFileSync(join(repoRoot, name), join(dest, name));
				}
			},
		},
	};
}
