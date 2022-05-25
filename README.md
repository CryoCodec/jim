# jim
![build](https://github.com/CryoCodec/jim/actions/workflows/build.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/CryoCodec/jim)](https://goreportcard.com/report/github.com/CryoCodec/jim)

Jim is a cli tool for connecting to different servers via SSH with ease. 
Jim is for you, if you're tired of remembering/entering server specific data for authentication. 

## How does it work?
You configure your authencation details in a json file and encrypt it with a master password. Jim spawns a Daemon process which will load the configuration file and after successful decryption hands out data to the jim client processes. The communication happens over an encrypted Unix Domain Socket. 

## Get Started

Head over to [releases](https://github.com/CryoCodec/jim/releases) and download the latest version for your platform. Currently only Mac and Linux is supported. Windows user's might give the WSL a chance. Decompress the download with `tar -zxvf jim-your-platform.tar.gz` and put the resulting folder on the PATH. The location of the files does not matter as long as the binaries are within the same directory. Now fire up your terminal and type: 
```bash
# This will give you some hints on how to get jim operational
jim doctor
```
Jim depends on SSH, SSHPass and pgrep and expects those to be available on the PATH. The doctor command will tell you, if everything is alright. Furthermore it will create a default json configuration file in ~/.jim/config.json. Have a look at the file and enter your server configuration. Once you're done, run: 
```bash
# This will check whether the config file can be parsed as jim config.
# Only if the validation did not show errors proceed.
jim validate path/to/your/config/file

# This will encrypt the file with a master password of your choice.
# Remember the password or put it in your favourite password manager, 
# since the password won't be stored anywhere by jim.
jim encrypt path/to/your/config/file
```

Congrats, you're ready to go. If you'd like to use a different location for your config file, set the environment variable JIM_CONFIG_FILE. Currently only the *list* and *connect* commands make use of this variable.  

If you'd like to adjust your configuration again just run `jim decrypt path/to/file` and the procedure from above. Note: at the moment the encrypt task does not delete the plaintext config file. You should consider its deletion ;)

Let's check if everything works as designed. Try to list all configured servers with: 
```bash
jim list
```
It will ask you for the master password and then print all entries in the config file. On subsequent requests the password is no longer necessary, as the daemon process is stateful and remembers the decrypted data. 

To connect to a server run: 
```bash
jim connect A Tag you have configured
```

The connect command will open a SSH connection to the server associated with the passed tag. The command supports fuzzy matching on tags. 

## Build
Just checkout this repository and run: 
```bash
make install-requirements-mac-x86_64 # or install-requirements-linux-x86_64
make protoc
make build
```
It is not necessary to build in the GOPATH. The build is tested on the latest GO v1.16.

## Shell Completions

jim offers shell completions for the connect command, showing valid candidates to connect to. To enable the shell completions execute

```bash
jim completion --help
```
 and follow the instructions. For better completion results when using tags with multiple spaces wrap the args in "", e.g. "Integration Webserver 1".

## Contribute
You miss a feature or found a bug? File an issue or open a Pull Request. 