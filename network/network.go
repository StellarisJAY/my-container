package network

import (
	"crypto/rand"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"path"
)

const (
	BridgeName = "my-container0"
	BridgeIP   = "172.40.0.1/16"
)

// SetupBridge 设置宿主机网桥
func SetupBridge() (*netlink.Bridge, error) {
	ptr, _ := netlink.LinkByName(BridgeName)
	var bridge0 *netlink.Bridge
	if br, ok := ptr.(*netlink.Bridge); ok {
		bridge0 = br
		return bridge0, nil
	} else {
		bridge0 = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name:   BridgeName,
				MTU:    1500,
				TxQLen: -1,
			},
		}
	}

	if err := netlink.LinkAdd(bridge0); err != nil {
		return nil, err
	}
	ipNet, _ := netlink.ParseIPNet(BridgeIP)
	if err := netlink.AddrAdd(bridge0, &netlink.Addr{
		IPNet: ipNet,
	}); err != nil {
		return nil, err
	}
	if err := netlink.LinkSetUp(bridge0); err != nil {
		return nil, err
	}

	return bridge0, nil
}

func CreateVeth(containerId string) error {
	vethName := getVethNamePrefix(containerId)
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:   vethName + "-ns", // 容器NS
			TxQLen: -1,
		},
		PeerName:         vethName + "-br", // peer 为宿主机bridge
		PeerHardwareAddr: createMACAddress(),
	}
	return netlink.LinkAdd(veth)
}

func CreateNetworkNamespace(containerId string) error {
	mntPath := getNetNsMountPoint(containerId)
	_ = util.CreateFileIfNotExist(mntPath)
	_ = util.CreateDirsIfNotExist([]string{path.Dir(mntPath)})
	// 当前的namespace fs
	fd, err := unix.Open("/proc/self/ns/net", unix.O_RDONLY|unix.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("can't open /proc/self/ns/net, error: %w", err)
	}
	defer unix.Close(fd)
	// 创建新的命名空间，当前进程进入新的namespace
	if err := unix.Unshare(unix.CLONE_NEWNET); err != nil {
		return fmt.Errorf("unable to unshare network namespace, error: %w", err)
	}
	// 新的namespace fd挂载到容器的mntPath
	if err := unix.Mount("/proc/self/ns/net", mntPath, "bind", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("unable to bind namespace to container mnt path, error: %w", err)
	}
	// 回到宿主机namespace
	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		return err
	}
	return nil
}

// SetupVethToBridge 在宿主机设置veth的主机端
func SetupVethToBridge(containerId string) error {
	bridge, _ := netlink.LinkByName(BridgeName)
	brVeth, _ := netlink.LinkByName(getVethNamePrefix(containerId) + "-br")
	if err := netlink.LinkSetMaster(brVeth, bridge); err != nil {
		return err
	}
	return netlink.LinkSetUp(brVeth)
}

// SetupVethInNamespace 在容器namespace配置veth的容器端
func SetupVethInNamespace(containerId string) error {
	log.Println("Setup veth in container namespace")
	mntPath := getNetNsMountPoint(containerId)
	fd, err := unix.Open(mntPath, unix.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open container namespace: %w", err)
	}
	defer unix.Close(fd)
	vethName := getVethNamePrefix(containerId) + "-ns"
	// 修改veth接口的命名空间
	veth, _ := netlink.LinkByName(vethName)
	if err := netlink.LinkSetNsFd(veth, fd); err != nil {
		return fmt.Errorf("unable to set veth's ns, error: %w", err)
	}
	// 进入容器的网络命名空间
	if err := unix.Setns(fd, unix.CLONE_NEWNET); err != nil {
		return fmt.Errorf("unable to setns to network namespace, error: %w", err)
	}
	veth, _ = netlink.LinkByName(vethName)
	ipNet, _ := netlink.ParseIPNet(getContainerIP())
	if err := netlink.AddrAdd(veth, &netlink.Addr{IPNet: ipNet}); err != nil {
		return fmt.Errorf("unable to add ip to veth, error: %w", err)
	}
	if err := netlink.LinkSetUp(veth); err != nil {
		return fmt.Errorf("unable to setup veth, error: %w", err)
	}
	return nil
}

func JoinNetworkNamespace(containerId string) error {
	mntPath := getNetNsMountPoint(containerId)
	fd, err := unix.Open(mntPath, unix.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to open container namespace: %w", err)
	}
	defer unix.Close(fd)

	return unix.Setns(fd, unix.CLONE_NEWNET)
}

func getVethNamePrefix(containerId string) string {
	return "veth" + containerId[:6]
}

func getNetNsMountPoint(containerId string) string {
	return common.NetNsBaseDir + containerId
}

func RemoveVeth(containerId, suffix string) {
	br, _ := netlink.LinkByName(getVethNamePrefix(containerId) + suffix)
	_ = netlink.LinkDel(br)
}

func getContainerIP() string {
	return "172.29.208.200/16"
}
func createMACAddress() net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)
	hw[0] = 0x02
	hw[1] = 0x42
	_, _ = rand.Read(hw[2:])
	return hw
}
