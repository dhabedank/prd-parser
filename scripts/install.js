#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const https = require('https');
const { createWriteStream, chmodSync, existsSync, mkdirSync } = require('fs');

const REPO = 'dhabedank/prd-parser';
const BIN_NAME = 'prd-parser';

// Map Node.js platform/arch to Go naming
function getPlatformInfo() {
  const platform = process.platform;
  const arch = process.arch;

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };

  const archMap = {
    x64: 'amd64',
    arm64: 'arm64',
  };

  const goPlatform = platformMap[platform];
  const goArch = archMap[arch];

  if (!goPlatform || !goArch) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }

  const ext = platform === 'win32' ? '.exe' : '';
  return { goPlatform, goArch, ext };
}

// Get the latest release version from GitHub
async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'api.github.com',
      path: `/repos/${REPO}/releases/latest`,
      headers: { 'User-Agent': 'prd-parser-installer' },
    };

    https.get(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        try {
          const release = JSON.parse(data);
          resolve(release.tag_name);
        } catch (e) {
          reject(new Error('Failed to parse GitHub response'));
        }
      });
    }).on('error', reject);
  });
}

// Download file with redirect support
function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const file = createWriteStream(dest);

    const request = (url) => {
      https.get(url, { headers: { 'User-Agent': 'prd-parser-installer' } }, (res) => {
        // Handle redirects
        if (res.statusCode === 302 || res.statusCode === 301) {
          request(res.headers.location);
          return;
        }

        if (res.statusCode !== 200) {
          reject(new Error(`Download failed with status ${res.statusCode}`));
          return;
        }

        res.pipe(file);
        file.on('finish', () => {
          file.close();
          resolve();
        });
      }).on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    };

    request(url);
  });
}

// Extract tar.gz archive
function extractTarGz(archive, dest) {
  execSync(`tar -xzf "${archive}" -C "${dest}"`, { stdio: 'inherit' });
}

// Extract zip archive
function extractZip(archive, dest) {
  if (process.platform === 'win32') {
    execSync(`powershell -Command "Expand-Archive -Path '${archive}' -DestinationPath '${dest}'"`, { stdio: 'inherit' });
  } else {
    execSync(`unzip -o "${archive}" -d "${dest}"`, { stdio: 'inherit' });
  }
}

async function main() {
  console.log('Installing prd-parser...');

  try {
    const { goPlatform, goArch, ext } = getPlatformInfo();
    console.log(`Platform: ${goPlatform}/${goArch}`);

    // Get latest version
    let version;
    try {
      version = await getLatestVersion();
      console.log(`Latest version: ${version}`);
    } catch (e) {
      console.log('Could not fetch latest version, using v0.1.0');
      version = 'v0.1.0';
    }

    // Determine archive format
    const archiveExt = goPlatform === 'windows' ? '.zip' : '.tar.gz';
    const archiveName = `prd-parser_${version.replace('v', '')}_${goPlatform}_${goArch}${archiveExt}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${archiveName}`;

    console.log(`Downloading from: ${downloadUrl}`);

    // Create temp directory
    const tempDir = path.join(__dirname, '..', '.temp');
    if (!existsSync(tempDir)) {
      mkdirSync(tempDir, { recursive: true });
    }

    const archivePath = path.join(tempDir, archiveName);
    await downloadFile(downloadUrl, archivePath);
    console.log('Download complete');

    // Extract
    console.log('Extracting...');
    if (archiveExt === '.zip') {
      extractZip(archivePath, tempDir);
    } else {
      extractTarGz(archivePath, tempDir);
    }

    // Move binary to bin directory
    const binDir = path.join(__dirname, '..', 'bin');
    if (!existsSync(binDir)) {
      mkdirSync(binDir, { recursive: true });
    }

    const srcBinary = path.join(tempDir, `${BIN_NAME}${ext}`);
    const destBinary = path.join(binDir, `${BIN_NAME}${ext}`);

    fs.copyFileSync(srcBinary, destBinary);

    // Make executable on Unix
    if (goPlatform !== 'windows') {
      chmodSync(destBinary, 0o755);
    }

    // Cleanup
    fs.rmSync(tempDir, { recursive: true, force: true });

    console.log(`✓ prd-parser installed successfully to ${destBinary}`);
    console.log('Run "prd-parser --help" to get started');

  } catch (error) {
    console.error('Installation failed:', error.message);
    console.error('');
    console.error('You can install manually:');
    console.error('  1. Download from https://github.com/dhabedank/prd-parser/releases');
    console.error('  2. Extract and move prd-parser to a directory in your PATH');
    process.exit(1);
  }
}

main();
