# legacybackupconverter

Convert legacy AntiRaid backup files to the new format. Intended to be used in both Rust FFI cases and normal human use.

## Project Structure

- ``iblfile``: Contains the parsing logic for the legacy backup files (minified to remove writing and encryption logic as only reading and decryption is needed). See [here](https://github.com/anti-raid/iblfile) for the original repository.
- ``main.go``: The main entry point for the conversion tool.