package scenario_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/scenario"
)

// Eleven call sites in this package generated a pack, and nine of them then
// decoded the result to get at its devices -- decoding bytes that Generate had
// already decoded and validated on the way to building the manifest. A pack is
// about 5 MB of YAML and yaml.v3 is roughly an order of magnitude slower under
// the race detector, so paying for that second decode once per test per pack
// ran the package past the 10-minute per-package timeout and make test could
// not complete on macOS. CI never saw it: ci.yml's macOS job runs -race over
// internal/truststore alone, and the full race suite runs on ubuntu (#2167).
//
// Generate now carries the config it built, so the fixture's methods are all a
// test needs.
//
// Generating was still most of the package's cost, because about thirty tests
// walk every pack and each walk generated all seven again: -race took 509 of
// CI's 600 s, and P-PACK-1's larger packs ran it past the limit (#2349). So a
// run generates each distinct request once. The result is shared, so
// it is read-only: a test that needs to change a generated config changes the
// request instead, which is a different key.
//
// The cache lives in one TestShippedPacks run rather than in package state, so
// every check that walks a pack is a subtest of it and takes the fixture:
// go test ./internal/scenario -run 'TestShippedPacks/<Check>'.
type packFixture struct {
	generations sync.Map
}

type packGeneration struct {
	once   sync.Once
	result scenario.Result
	err    error
}

func TestShippedPacks(t *testing.T) {
	fixture := &packFixture{}
	checks := []struct {
		name string
		run  func(*testing.T, *packFixture)
	}{
		{"AccessPointSerialsAreUnique", testAccessPointSerialsAreUnique},
		{"CampusCollapsesTheCore", testCampusCollapsesTheCore},
		{"CompactNameShape", testCompactNameShape},
		{"DeclaredLinksAreDiscoverable", testDeclaredLinksAreDiscoverable},
		{"DevicesReportToTheirOwnSiteCollector", testDevicesReportToTheirOwnSiteCollector},
		{"EachFindingReachesTheAuthoredConfig", testEachFindingReachesTheAuthoredConfig},
		{"EachVerticalRunsADataCentreTierAtItsFirstSite", testEachVerticalRunsADataCentreTierAtItsFirstSite},
		{"EachVerticalRunsItsOtherWorkloadSystems", testEachVerticalRunsItsOtherWorkloadSystems},
		{"EachVerticalRunsItsWorkloadServer", testEachVerticalRunsItsWorkloadServer},
		{"EndpointUtilizationVariesAcrossDevices", testEndpointUtilizationVariesAcrossDevices},
		{"EveryAccessSwitchOffersASparePortPool", testEveryAccessSwitchOffersASparePortPool},
		{"EveryPackPointsItsDevicesAtACollector", testEveryPackPointsItsDevicesAtACollector},
		{"EveryPackSiteCarriesAUPSThatAnswersUPSMIB", testEveryPackSiteCarriesAUPSThatAnswersUPSMIB},
		{"EveryPackSiteCarriesItsNASAndRoomSystems", testEveryPackSiteCarriesItsNASAndRoomSystems},
		{"EverySyslogSenderCanRouteToItsCollector", testEverySyslogSenderCanRouteToItsCollector},
		{"GeneratedAccessPointRadiosAnswerTheCaptureShape", testGeneratedAccessPointRadiosAnswerTheCaptureShape},
		{"GeneratedAccessPointsAuthorTheirRadios", testGeneratedAccessPointsAuthorTheirRadios},
		{"GeneratedAccessPointsJoinTheirSiteController", testGeneratedAccessPointsJoinTheirSiteController},
		{"GeneratedAccessPointsServeOneSourcePerDot11Object", testGeneratedAccessPointsServeOneSourcePerDot11Object},
		{"GeneratedDeviceNamesAreUnique", testGeneratedDeviceNamesAreUnique},
		{"GeneratedPoolCompilesToOneNetwork", testGeneratedPoolCompilesToOneNetwork},
		{"HospitalAccessLayerStaysDualHomed", testHospitalAccessLayerStaysDualHomed},
		{"HospitalMixKeepsSignatureDevicesRareAtScale", testHospitalMixKeepsSignatureDevicesRareAtScale},
		{"HospitalPackAccessPointsAdvertiseMED", testHospitalPackAccessPointsAdvertiseMED},
		{"HospitalPackAccessSwitchesAuthorAPoEBudget", testHospitalPackAccessSwitchesAuthorAPoEBudget},
		{"HospitalPackGeneratesPhilipsAndGEPatientMonitors", testHospitalPackGeneratesPhilipsAndGEPatientMonitors},
		{"HospitalPackHasMEDEndpoints", testHospitalPackHasMEDEndpoints},
		{"HospitalPackPoEBudgetCoversItsPoweredDevices", testHospitalPackPoEBudgetCoversItsPoweredDevices},
		{"MEDEndpointsDoNotCollideWithWiredEndpoints", testMEDEndpointsDoNotCollideWithWiredEndpoints},
		{"ManifestCarriesTheAuthoredFindings", testManifestCarriesTheAuthoredFindings},
		{"ManifestDeclaresItsSchemaVersion", testManifestDeclaresItsSchemaVersion},
		{
			"ManifestExpectedObservationsFollowTheAuthoredConfig",
			testManifestExpectedObservationsFollowTheAuthoredConfig,
		},
		{"ManifestIdentityIsTheReproducibleInput", testManifestIdentityIsTheReproducibleInput},
		{"ManifestNeverCarriesPhysicalBinding", testManifestNeverCarriesPhysicalBinding},
		{"ManifestOmitsObservationsNothingServes", testManifestOmitsObservationsNothingServes},
		{"ManifestRecordsInterfaceTruth", testManifestRecordsInterfaceTruth},
		{
			"ManifestTimingFollowsProtocolAdvertisementIntervals",
			testManifestTimingFollowsProtocolAdvertisementIntervals,
		},
		{"ManufacturingAccessLayerIsARing", testManufacturingAccessLayerIsARing},
		{"ManufacturingRingJoinsDistributionTwice", testManufacturingRingJoinsDistributionTwice},
		{"NoPackNamesTheTesterInAnInterfaceDescription", testNoPackNamesTheTesterInAnInterfaceDescription},
		{"NoStaticAddressFallsInADHCPPool", testNoStaticAddressFallsInADHCPPool},
		{"PackCoreDistributionUplinksAreBundles", testPackCoreDistributionUplinksAreBundles},
		{"PackCoresRouteIPv6OnEveryVLAN", testPackCoresRouteIPv6OnEveryVLAN},
		{"PackDeviceRoleMix", testPackDeviceRoleMix},
		{"PackWirelessNetworkIsRelayedToTheSiteDHCPServer", testPackWirelessNetworkIsRelayedToTheSiteDHCPServer},
		{"PackWirelessStationsAreTheDevicesTheirAPsReport", testPackWirelessStationsAreTheDevicesTheirAPsReport},
		{"PacksServeDICOMOnModalitiesAndPACS", testPacksServeDICOMOnModalitiesAndPACS},
		{"PacksServeEtherNetIPOnPLCs", testPacksServeEtherNetIPOnPLCs},
		{"PacksServeFTPOnTheHPPrinters", testPacksServeFTPOnTheHPPrinters},
		{"PacksServeHL7OnClinicalSystems", testPacksServeHL7OnClinicalSystems},
		{"PacksServeJetDirectOnThePrinters", testPacksServeJetDirectOnThePrinters},
		{"PacksServeModbusOnPLCs", testPacksServeModbusOnPLCs},
		{"PacksServeONVIFOnTheCameras", testPacksServeONVIFOnTheCameras},
		{"PacksServeSIPOnThePhones", testPacksServeSIPOnThePhones},
		{"PacksServeSSHWithoutAnAccount", testPacksServeSSHWithoutAnAccount},
		{"PersonalComputersAnswerNoSNMP", testPersonalComputersAnswerNoSNMP},
		{"RetailStoreChainsItsLanes", testRetailStoreChainsItsLanes},
		{"ServiceProviderRingsItsPOPAndKeepsTheSpine", testServiceProviderRingsItsPOPAndKeepsTheSpine},
		{"TesterAttachesToAnAccessPortPool", testTesterAttachesToAnAccessPortPool},
		{"TheCollectorDoesNotReportToItself", testTheCollectorDoesNotReportToItself},
		{"VerticalPacksGenerateDistinctEndpointProfiles", testVerticalPacksGenerateDistinctEndpointProfiles},
		{"WritePackDeviceRoles", testWritePackDeviceRoles},
		{"WritePackParity", testWritePackParity},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			check.run(t, fixture)
		})
	}
}

// result generates a pack, failing the test if it cannot.
func (f *packFixture) result(t *testing.T, pack scenario.Pack) scenario.Result {
	t.Helper()

	return f.request(t, pack.ID, pack.Request)
}

// request generates request once per fixture, failing the test if it cannot.
// label names the request in a failure.
func (f *packFixture) request(t *testing.T, label string, request scenario.Request) scenario.Result {
	t.Helper()

	key, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("key %s: %v", label, err)
	}
	entry, _ := f.generations.LoadOrStore(string(key), &packGeneration{})
	generation, _ := entry.(*packGeneration)
	generation.once.Do(func() {
		generation.result, generation.err = scenario.Generate(request)
	})
	if generation.err != nil {
		t.Fatalf("generate %s: %v", label, generation.err)
	}

	return generation.result
}

// config is the generated pack's runtime config.
func (f *packFixture) config(t *testing.T, pack scenario.Pack) *config.Config {
	t.Helper()

	return f.result(t, pack).Config
}

// configByID is the runtime config of the shipped pack named id.
func (f *packFixture) configByID(t *testing.T, id string) *config.Config {
	t.Helper()
	for _, pack := range scenario.Packs() {
		if pack.ID == id {
			return f.config(t, pack)
		}
	}
	t.Fatalf("no pack %q", id)

	return nil
}
