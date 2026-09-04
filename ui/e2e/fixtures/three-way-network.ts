/**
 * One network, three ways of authoring it.
 *
 * P1b-3's decision is that the YAML file, the device editor and the wizard can
 * each author everything the daemon runs. This fixture is what makes that
 * checkable end to end: each route produces this network, and each asserts the
 * same expectation against the *running session* rather than against the form
 * that produced it.
 *
 * Deliberately small. The clinic scenario in docs/examples is the reference for
 * documentation and for the contract tests, but it authors SSH, and a device
 * with `ssh.password_env` cannot start unless that variable is set in the
 * daemon's environment -- which would couple this suite to the launcher's env
 * rather than to the authoring surfaces it is testing.
 */

/** The interface the dry-run daemon binds; matches the attachment policy the
 * E2E launcher approves. */
export const SIM_INTERFACE = process.env.E2E_SIM_INTERFACE ?? 'e2e-dry-run0';

/** The attachment the config declares, and the binding used to start it. */
export const ATTACHMENT = { name: 'tester', mode: 'access', accessVlan: 200 } as const;

export const NETWORK = { name: 'e2e-lan', subnet: '10.77.0.0/24' } as const;

/** What every route must end up having authored. */
export interface ExpectedDevice {
  hostname: string;
  type: string;
  mac: string;
  address: string;
  interfaceName: string;
  /** Neighbour on the other end of this device's single link. */
  linkedTo: string;
  snmpCommunity: string;
}

export const EXPECTED_DEVICES: readonly ExpectedDevice[] = [
  {
    hostname: 'e2e-rtr-01',
    type: 'router',
    mac: '02:00:00:00:77:01',
    address: '10.77.0.1/24',
    interfaceName: 'Ethernet1/1',
    linkedTo: 'e2e-sw-01',
    snmpCommunity: 'e2e_public',
  },
  {
    hostname: 'e2e-sw-01',
    type: 'switch',
    mac: '02:00:00:00:77:02',
    address: '10.77.0.2/24',
    interfaceName: 'Ethernet1/1',
    linkedTo: 'e2e-rtr-01',
    snmpCommunity: 'e2e_public',
  },
];

const device = (spec: ExpectedDevice) => `  - name: ${spec.hostname}
    type: ${spec.type}
    mac: "${spec.mac}"
    icmp:
      enabled: true
    snmp_agent:
      enabled: true
      community: ${spec.snmpCommunity}
      sysname: ${spec.hostname}
    interfaces:
      - name: ${spec.interfaceName}
        type: ethernet
        network: ${NETWORK.name}
        address: ${spec.address}
    trunk_ports:
      - interface: ${spec.interfaceName}
        native_vlan: 1
        remote_device: ${spec.linkedTo}
        remote_interface: ${spec.interfaceName}
`;

/** The canonical YAML: what route 1 uploads and route 2 edits. */
export const NETWORK_YAML = `networks:
  - name: ${NETWORK.name}
    subnet: ${NETWORK.subnet}

attachments:
  - name: ${ATTACHMENT.name}
    connect: ${NETWORK.name}

devices:
${EXPECTED_DEVICES.map(device).join('\n')}`;
