go-md3
======

MD3 (a la Quake 3) model loader for Go. This covers MD3 support as is used in Quake 3. It includes the `md3` package for reading MD3 data and the `go-md3` commandline tool for briefly inspecting MD3 files and converting them to OBJ files.


go-md3 tool
-----------

The go-md3 tool has three modes and a few options that affect only specific modes. The modes can be specified via `-mode=[name]` (or `-mode name`). The default mode is `spec`.

- `spec`

    Has no additional options. Displays a summary of the contents of any provided MD3 files, including surfaces, skins, frame counts, tag names, and so on.

- `convert`

    Converts provided MD3 files to OBJ files. Each frame is written as a separate OBJ file, named as such: `<basename>+<frameNumber>.obj`. Takes a few options:

    - `-flipUVs=[on|off]` — if on, the V texture coordinates will be flipped by writing them out as `(1.0 - V)`. This is helpful if you work in a system that doesn't play nice with OpenGL's texture coordinate space. Defaults to on.

    - `-swapYZ=[on|off]` — if on, the Y and Z axes are swapped for vertices and normals. Defaults to on.

    - `-o=path/to/output` — sets the output directory for OBJ files. Defaults to the current directory (`.`).

- `view`

    Currently unimplemented and will result in a panic. Intended for viewing MD3 files with a given set of textures -- won't include a proper emulation of the Quake 3 shader system and such.


License
-------

go-md3 is licensed under the simplified two-clause BSD license, found in the accompanying LICENSE file.
