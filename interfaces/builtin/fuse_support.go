// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2017 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin

import (
	"fmt"

	"github.com/snapcore/snapd/interfaces"
	"github.com/snapcore/snapd/interfaces/apparmor"
	"github.com/snapcore/snapd/interfaces/seccomp"
	"github.com/snapcore/snapd/interfaces/udev"
	"github.com/snapcore/snapd/release"
)

const fuseSupportSummary = `allows access to the FUSE file system`

const fuseSupportBaseDeclarationSlots = `
  fuse-support:
    allow-installation:
      slot-snap-type:
        - core
    deny-auto-connection: true
`

const fuseSupportConnectedPlugSecComp = `
# Description: Can run a FUSE filesystem. Unprivileged fuse mounts are
# not supported at this time.

mount
`

const fuseSupportConnectedPlugAppArmor = `
# Description: Can run a FUSE filesystem. Unprivileged fuse mounts are
# not supported at this time.

# Allow communicating with fuse kernel driver
# https://www.kernel.org/doc/Documentation/filesystems/fuse.txt
/dev/fuse rw,

# Required for mounts
capability sys_admin,

# Allow mounts to our snap-specific writable directories
# Note 1: fstype is 'fuse.<command>', eg 'fuse.sshfs'
# Note 2: due to LP: #1612393 - @{HOME} can't be used in mountpoint
# Note 3: local fuse mounts of filesystem directories are mediated by 
#         AppArmor. The actual underlying file in the source directory is
#         mediated, not the presentation layer of the target directory, so
#         we can safely allow all local mounts to our snap-specific writable
#         directories.
# Note 4: fuse supports a lot of different mount options, and applications
#         are not obligated to use fusermount to mount fuse filesystems, so
#         be very strict and only support the default (rw,nosuid,nodev) and
#         read-only.
mount fstype=fuse.* options=(ro,nosuid,nodev) ** -> /home/*/snap/@{SNAP_NAME}/@{SNAP_REVISION}/{,**/},
mount fstype=fuse.* options=(rw,nosuid,nodev) ** -> /home/*/snap/@{SNAP_NAME}/@{SNAP_REVISION}/{,**/},
mount fstype=fuse.* options=(ro,nosuid,nodev) ** -> /var/snap/@{SNAP_NAME}/@{SNAP_REVISION}/{,**/},
mount fstype=fuse.* options=(rw,nosuid,nodev) ** -> /var/snap/@{SNAP_NAME}/@{SNAP_REVISION}/{,**/},

# Explicitly deny reads to /etc/fuse.conf. We do this to ensure that
# the safe defaults of fuse are used (which are enforced by our mount
# rules) and not system-specific options from /etc/fuse.conf that
# may conflict with our mount rules.
deny /etc/fuse.conf r,

# Allow read access to the fuse filesystem
/sys/fs/fuse/ r,
/sys/fs/fuse/** r,

# Unprivileged fuser mounts must use the setuid helper in the core snap
# (not currently available, so don't include in policy at this time).
#/{,usr/}bin/fusermount ixr,
`

const fuseSupportConnectedPlugUdev = `
# This file contains udev rules for FUSE filesystem.
#
# Do not edit this file, it will be overwritten on updates

KERNEL=="fuse", TAG+="%s"
`

// The type for fuse-support interface
type fuseSupportInterface struct{}

// Getter for the name of the fuse support interface
func (iface *fuseSupportInterface) Name() string {
	return "fuse-support"
}

func (iface *fuseSupportInterface) MetaData() interfaces.MetaData {
	return interfaces.MetaData{
		Summary:              fuseSupportSummary,
		ImplicitOnCore:       true,
		ImplicitOnClassic:    !(release.ReleaseInfo.ID == "ubuntu" && release.ReleaseInfo.VersionID == "14.04"),
		BaseDeclarationSlots: fuseSupportBaseDeclarationSlots,
	}
}

func (iface *fuseSupportInterface) String() string {
	return iface.Name()
}

// Check validity of the defined slot
func (iface *fuseSupportInterface) SanitizeSlot(slot *interfaces.Slot) error {
	// Does it have right type?
	if iface.Name() != slot.Interface {
		panic(fmt.Sprintf("slot is not of interface %q", iface))
	}
	return nil
}

// Checks and possibly modifies a plug
func (iface *fuseSupportInterface) SanitizePlug(plug *interfaces.Plug) error {
	if iface.Name() != plug.Interface {
		panic(fmt.Sprintf("plug is not of interface %q", iface))
	}
	// Currently nothing is checked on the plug side
	return nil
}

func (iface *fuseSupportInterface) AppArmorConnectedPlug(spec *apparmor.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	spec.AddSnippet(fuseSupportConnectedPlugAppArmor)
	return nil
}

func (iface *fuseSupportInterface) SecCompConnectedPlug(spec *seccomp.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	spec.AddSnippet(fuseSupportConnectedPlugSecComp)
	return nil
}

func (iface *fuseSupportInterface) UDevConnectedPlug(spec *udev.Specification, plug *interfaces.Plug, plugAttrs map[string]interface{}, slot *interfaces.Slot, slotAttrs map[string]interface{}) error {
	for appName := range plug.Apps {
		tag := udevSnapSecurityName(plug.Snap.Name(), appName)
		spec.AddSnippet(fmt.Sprintf(fuseSupportConnectedPlugUdev, tag))
	}
	return nil
}

func (iface *fuseSupportInterface) AutoConnect(*interfaces.Plug, *interfaces.Slot) bool {
	// Allow what is allowed in the declarations
	return true
}

func init() {
	registerIface(&fuseSupportInterface{})
}
