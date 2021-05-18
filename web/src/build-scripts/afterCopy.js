const fs = require('fs');
const path = require('path');

const WindowsPlatform = "win32"

module.exports = function(buildPath, electronVersion, platform, arch, callback) {
    let sourceDir = path.join(__dirname, "../../../out") // Go output directory
    let platformBinary = platform
    if (platform === WindowsPlatform) {
        platformBinary = "windows"
    }
    let platformSources = fs.readdirSync(sourceDir).filter(f => f.includes(`-${platformBinary}-x86_64`))
    if (platformSources.length === 0) {
        throw Error(`No binary found for platform: ${platform}`)
    }
    let source = path.join(sourceDir, platformSources[0]);
    console.log(`\nCopying Sage Go binary to resources dir: ${source}`)
    let destination = path.join(buildPath, '../sage-server'); // one directory up is the resources directory
    if (platform === WindowsPlatform) {
        destination += ".exe"
    }

    fs.copyFileSync(source, destination)

    callback();
}
