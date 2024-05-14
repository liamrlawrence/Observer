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
-   **`rebuild_delay`** _(optional)_: The delay (in milliseconds) after a file is changed before rebuilding. If omitted, the default is 500ms.
-   **`init_commands`** _(optional)_: An array that contains a list of commands that should be run once on startup. This can be especially useful when working with Docker containers.
-   **`watchers`** _(required)_: An array that contains configurations for different sets of files to watch. Each watcher can be customized with its own set of rules and commands.

### Watcher Configuration
-   **`label`** _(optional, **recommended**)_: A label used when printing information to the console. If omitted, a default label will be generated, however it is not the most user friendly.

-   **`extensions`** _(required)_: An array of file extensions to watch.

-   **`include_dirs`** _(optional)_: An array of directories to watch. If omitted, the tool defaults to the current working directory (`"."`).

-   **`ignore_dirs`** _(optional)_: An array of directories to ignore.

-   **`include_patterns`** _(optional)_: An array of regular expression patterns. Paths matching these patterns will be watched.

-   **`ignore_patterns`** _(optional)_: An array of regular expression patterns. Paths matching these patterns will be ignored.

-   **`build_command`** _(required)_: The command to execute when changes are detected that meet the criteria. This command is typically a build or compile command.

-   **`run_command`** _(optional)_: An additional command that can be run after the `build_command`. This could be used to execute a program or script.

-   **`debug`** _(optional)_: A boolean to toggle additional diagnostic information.

Here is an example of a configuration file:
```json
{
  "init_commands": [
    "cp -r node_modules_cache/* node_modules/"
  ],
  "watchers": [
    {
      "label": "[Go]",
      "extensions": [".go", ".templ"],
      "include_dirs": ["cmd", "internal"],
      "ignore_patterns": ["_templ\\.go"],
      "build_command": "make go",
      "run_command": "./build/main"
    },
    {
      "label": "[Web]",
      "extensions": [".ts"],
      "include_dirs": ["static"],
      "build_command": "npm run build"
    }
  ]
}
```

#### Optional Configuration Flags
You can specify custom flags at runtime to configure the behavior of the application. Here are the available flags:

- `-config`: This flag allows you to specify a custom configuration file.
    ```bash
    observer -config path/to/your/config.json
    ```
  If omitted, Observer defaults to using `observer.config.json` from the current working directory.


- `-debug`: Setting this flag to true activates debug mode for all watchers, regardless of what is specified in their configuration. In debug mode, additional diagnostic information is logged.
    ```bash
    observer -debug=true
    ```


## License
This project is licensed under the MIT License.

