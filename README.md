# Observer

Observer is a Golang project designed to enhance development workflows by automating the process of watching file changes within a directory, and then executing build and run commands for hot reloading.

## Features
- **File Monitoring**: Automatically detects changes in specified files based on extensions, directories, and regex patterns.
- **Configurable**: Allows for detailed configuration to specify which files to watch and what commands to trigger.
- **Hot Reloading**: Executes build and run commands automatically on file change, facilitating hot reloading during development.

## Installation
Via `go install` (Recommended)

With go 1.22 or higher:
```bash
go install github.com/liamrlawrence/observer/cmd/observer@latest
```

## Configuration
The Observer tool uses a JSON configuration file to define how it should monitor and respond to file changes in directories. Below is a detailed description of each configuration parameter available in `observer.config.json`.

### Top-Level Configuration
-   **`refresh_rate`** _(optional)_: The refresh rate (in milliseconds) defines how frequently the tool checks for file changes. This value impacts how quickly changes are detected and processed. The default value is 500ms.

### Watchers Configuration
The `watchers` array contains configurations for different sets of files to watch. Each watcher can be customized with its own set of rules and commands.

-   **`extensions`** _(required)_: An array of file extensions to watch.

-   **`include_dirs`** _(optional)_: An array of directories to watch. If omitted, the tool defaults to the current working directory (`"."`).

-   **`ignore_dirs`** _(optional)_: An array of directories to ignore.

-   **`include_patterns`** _(optional)_: An array of regular expression patterns. Paths matching these patterns will be watched.

-   **`ignore_patterns`** _(optional)_: An array of regular expression patterns. Paths matching these patterns will be ignored.

-   **`build_command`** _(required)_: The command to execute when changes are detected that meet the criteria. This command is typically a build or compile command.

-   **`run_command`** _(optional)_: An additional command that can be run after the `build_command`. This could be used to execute a program or script.

Here is an example of a configuration file:
```json
{
  "watchers": [
    {
      "extensions": [".go", ".templ"],
      "include_dirs": ["cmd", "internal"],
      "ignore_patterns": ["_templ\\.go"],
      "build_command": "make go",
      "run_command": "./build/main"
    },
    {
      "extensions": [".ts"],
      "include_dirs": ["static"],
      "build_command": "npm run build"
    }
  ]
}
```

#### Optional Configuration Flag
You can specify a custom configuration file at runtime using the `-config` flag. For example:
```bash
observer -config path/to/your/config.json
```
If the `-config` flag is not used, Observer defaults to using `observer.config.json` from the current working directory.


## License
This project is licensed under the MIT License.

