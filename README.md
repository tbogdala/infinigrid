Infinigrid: Escape (v0.1.0 DEVELOPMENT)
=======================================

A prototype game of a procedural infinite 'runner' where the player controls a ship
and has to avoid flying objects.

It exists as a larger scale 'example' for my graphics engine [fizzle][fizzle] as well
as my OpenVR wrapper [openvr-go][ovrgo].


Installation
============

Currently binaries must be built by source which will require [Go][golang] to be installed. 
Dependencies are handled by the [dep][godep] tool, so [Go][golang] 1.8+ will be required.

This repository does use [git lfs][gitlfs] so you will need to have that installed before
cloning the repo to avoid complication. Also, make sure to run `git lfs install` to make
sure the git hooks are installed! A symptom of not getting [git lfs][gitlfs] right is that
the binary files in the `assets` folders will all be pointers and not actual textures and models.

```bash
go get github.com/tbogdala/infinigrid
cd $GOHOME/src/github.com/tbogdala/infinigrid
dep ensure
go build
cp vendor/github.com/tbogdala/openvr-go/vendored/openvr/bin/<$PLATFORM>/openvr_api.dll .
```

Replace `<$PLATFORM>` above with `win64` or `linux64` or whatever you need to match
the system you're building on.


Running
========

After building, the game can be run by executing the binary:

```bash
./infinigrid
```

The keyboard keys are mapped to WASD for ship movement.

---

If you wish to play in VR mode, append the `-vr` flag:

```bash
./infinigrid -vr
```

The vive wand is tilted to control which direction the ship should move in; pointing
the vive controller straight up should be 'neutral'. The menu button can be pressed
while playing to reset the head height for the HMD.

Once the player lost, pressing the menu button should restart the game.


LICENSE
========

This game source code is released under the GPL v3 license; see the LICENSE file for more detail.

The art assets in the `assets` and `assets-src` folder are released under the [Creative Commons
Attribution-ShareAlike 4.0 (CC BY-SA 4.0)][ccbysa4] license.


[golang]: https://golang.org/
[fizzle]: https://github.com/tbogdala/fizzle
[ovrgo]: https://github.com/tbogdala/openvr-go
[godep]: https://github.com/golang/dep
[ccbysa4]: https://creativecommons.org/licenses/by-sa/4.0/
[gitlfs]: https://git-lfs.github.com/