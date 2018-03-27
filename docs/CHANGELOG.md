---
title: Changelog | Voyager
description: Changelog
menu:
  product_voyager_6.0.0:
    identifier: changelog-voyager
    name: Changelog
    parent: welcome
    weight: 10
product_name: voyager
menu_name: product_voyager_6.0.0
section_menu_id: welcome
url: /products/voyager/6.0.0/welcome/changelog/
aliases:
  - /products/voyager/6.0.0/CHANGELOG/
---

# Change Log

## [6.0.0](https://github.com/appscode/voyager/tree/6.0.0) (2018-03-13)
[Full Changelog](https://github.com/appscode/voyager/compare/6.0.0-rc.2...6.0.0)

**Fixed bugs:**

- TCP Ingress: invalid memory address or nil pointer dereference [\#906](https://github.com/appscode/voyager/issues/906)
- Voyager 6.0.0 on GKE 1.8.5:  Failed to list \*v1beta1.Ingress: unstructured cannot convert field labels [\#889](https://github.com/appscode/voyager/issues/889)
- Run service monitor informer in its own go routine. [\#929](https://github.com/appscode/voyager/pull/929) ([tamalsaha](https://github.com/tamalsaha))
- Various fixes and improved logging [\#928](https://github.com/appscode/voyager/pull/928) ([tamalsaha](https://github.com/tamalsaha))
- Use user provided cookie name for default backend [\#920](https://github.com/appscode/voyager/pull/920) ([tamalsaha](https://github.com/tamalsaha))
- Fixed ingress finalizer [\#917](https://github.com/appscode/voyager/pull/917) ([diptadas](https://github.com/diptadas))
- Detect change when deletion timestamp is set for Ingress [\#916](https://github.com/appscode/voyager/pull/916) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Support LB type in Openstack [\#930](https://github.com/appscode/voyager/issues/930)
- Deployment model of voyager a bit overcomplex? [\#924](https://github.com/appscode/voyager/issues/924)
- HTTP to HTTPS redirect [\#923](https://github.com/appscode/voyager/issues/923)
- OpenStack support [\#669](https://github.com/appscode/voyager/issues/669)
- Expose HAProxy config template var w/ Voyager deployment.spec.replicas [\#517](https://github.com/appscode/voyager/issues/517)
- Improve AWS support [\#163](https://github.com/appscode/voyager/issues/163)
- Use alpine as the base image for haproxy [\#108](https://github.com/appscode/voyager/issues/108)
- Truly Seamless Reloads with HAProxy [\#89](https://github.com/appscode/voyager/issues/89)

**Merged pull requests:**

- Update docs for 6.0.0 [\#932](https://github.com/appscode/voyager/pull/932) ([tamalsaha](https://github.com/tamalsaha))
- Document how to create internal LB in openstack [\#931](https://github.com/appscode/voyager/pull/931) ([tamalsaha](https://github.com/tamalsaha))
- Fix typo in README [\#927](https://github.com/appscode/voyager/pull/927) ([shaneog](https://github.com/shaneog))
- Update overview.md [\#926](https://github.com/appscode/voyager/pull/926) ([bewiwi](https://github.com/bewiwi))
- Add "New to Voyager" header [\#922](https://github.com/appscode/voyager/pull/922) ([tamalsaha](https://github.com/tamalsaha))
- Add --purge flag [\#921](https://github.com/appscode/voyager/pull/921) ([tamalsaha](https://github.com/tamalsaha))
- Make headerRule, rewriteRule plural [\#919](https://github.com/appscode/voyager/pull/919) ([tamalsaha](https://github.com/tamalsaha))
- Make it clear that installer is a single command [\#915](https://github.com/appscode/voyager/pull/915) ([tamalsaha](https://github.com/tamalsaha))

## [6.0.0-rc.2](https://github.com/appscode/voyager/tree/6.0.0-rc.2) (2018-03-05)
[Full Changelog](https://github.com/appscode/voyager/compare/6.0.0-rc.1...6.0.0-rc.2)

**Merged pull requests:**

- Update docs that --rbac is default on [\#914](https://github.com/appscode/voyager/pull/914) ([tamalsaha](https://github.com/tamalsaha))
- Enable RBAC by default in installer [\#913](https://github.com/appscode/voyager/pull/913) ([tamalsaha](https://github.com/tamalsaha))
- Fix installer [\#912](https://github.com/appscode/voyager/pull/912) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 6.0.0-rc.2 [\#911](https://github.com/appscode/voyager/pull/911) ([tamalsaha](https://github.com/tamalsaha))
- Stop using field selector in haproxy controller [\#910](https://github.com/appscode/voyager/pull/910) ([tamalsaha](https://github.com/tamalsaha))
- Update chart to match RBAC best practices for charts [\#909](https://github.com/appscode/voyager/pull/909) ([tamalsaha](https://github.com/tamalsaha))
- Add checks to installer script [\#908](https://github.com/appscode/voyager/pull/908) ([tamalsaha](https://github.com/tamalsaha))
- Cleanup admission webhook [\#907](https://github.com/appscode/voyager/pull/907) ([tamalsaha](https://github.com/tamalsaha))
- Update changelog for 6.0.0-rc.1 [\#905](https://github.com/appscode/voyager/pull/905) ([tamalsaha](https://github.com/tamalsaha))

## [6.0.0-rc.1](https://github.com/appscode/voyager/tree/6.0.0-rc.1) (2018-02-28)
[Full Changelog](https://github.com/appscode/voyager/compare/6.0.0-rc.0...6.0.0-rc.1)

**Implemented enhancements:**

- Source IP affinity [\#759](https://github.com/appscode/voyager/issues/759)

**Fixed bugs:**

- basic auth remove on upgrade to 5.0.0-rc11 [\#873](https://github.com/appscode/voyager/issues/873)
- whitelist did not work [\#866](https://github.com/appscode/voyager/issues/866)
- Update voyager docs [\#50](https://github.com/appscode/voyager/issues/50)

**Closed issues:**

- Update Prometheus integration [\#893](https://github.com/appscode/voyager/issues/893)
- Disabling HSTS - doesn't work [\#881](https://github.com/appscode/voyager/issues/881)
- Upgrade from 5.0.0-rc.11 to 6.0.0-rc.0 [\#876](https://github.com/appscode/voyager/issues/876)
- AWS ELB Proxy IP forwarded for occurs errors  [\#749](https://github.com/appscode/voyager/issues/749)
- How to use voyager instead of kubernetes nginx ingress controller [\#742](https://github.com/appscode/voyager/issues/742)
- RBAC for voyager [\#732](https://github.com/appscode/voyager/issues/732)
- Document default mode does not work for minikube [\#545](https://github.com/appscode/voyager/issues/545)
- Document how to use Host IP as external IP in minikube for LoadBalancer type Service [\#511](https://github.com/appscode/voyager/issues/511)
- Document RBAC setup on installer page. [\#508](https://github.com/appscode/voyager/issues/508)
- Document external-dns configuration [\#355](https://github.com/appscode/voyager/issues/355)
- Document why each ingress creates a new HAProxy in voyager [\#331](https://github.com/appscode/voyager/issues/331)

**Merged pull requests:**

- Prepare docs for 6.0.0-rc.1 [\#904](https://github.com/appscode/voyager/pull/904) ([tamalsaha](https://github.com/tamalsaha))
- Fix service name in chart [\#903](https://github.com/appscode/voyager/pull/903) ([tamalsaha](https://github.com/tamalsaha))
- Update links to latest release [\#901](https://github.com/appscode/voyager/pull/901) ([tamalsaha](https://github.com/tamalsaha))
- Support --enable-admission-webhook=false [\#900](https://github.com/appscode/voyager/pull/900) ([tamalsaha](https://github.com/tamalsaha))
- Support multiple webhooks of same apiversion [\#899](https://github.com/appscode/voyager/pull/899) ([tamalsaha](https://github.com/tamalsaha))
- Sync chart to stable charts repo [\#898](https://github.com/appscode/voyager/pull/898) ([tamalsaha](https://github.com/tamalsaha))
- Document Prometheus integration [\#896](https://github.com/appscode/voyager/pull/896) ([tamalsaha](https://github.com/tamalsaha))
- Improve docs [\#895](https://github.com/appscode/voyager/pull/895) ([tamalsaha](https://github.com/tamalsaha))
- Update haproxy exporter [\#894](https://github.com/appscode/voyager/pull/894) ([tamalsaha](https://github.com/tamalsaha))
- Document user facing RBAC roles [\#892](https://github.com/appscode/voyager/pull/892) ([tamalsaha](https://github.com/tamalsaha))
- Skip generating UpdateStatus method [\#887](https://github.com/appscode/voyager/pull/887) ([tamalsaha](https://github.com/tamalsaha))
- Delete internal types [\#886](https://github.com/appscode/voyager/pull/886) ([tamalsaha](https://github.com/tamalsaha))
- Use official code generator scripts [\#885](https://github.com/appscode/voyager/pull/885) ([tamalsaha](https://github.com/tamalsaha))
- Use HAProxy 1.8.5 [\#884](https://github.com/appscode/voyager/pull/884) ([tamalsaha](https://github.com/tamalsaha))
- Move node selector to Ingress spec [\#883](https://github.com/appscode/voyager/pull/883) ([tamalsaha](https://github.com/tamalsaha))
- Only check NodePort if provided [\#880](https://github.com/appscode/voyager/pull/880) ([tamalsaha](https://github.com/tamalsaha))
- Create user facing aggregate roles [\#879](https://github.com/appscode/voyager/pull/879) ([tamalsaha](https://github.com/tamalsaha))
- Use rbac/v1 api [\#878](https://github.com/appscode/voyager/pull/878) ([tamalsaha](https://github.com/tamalsaha))
- Use github.com/pkg/errors [\#877](https://github.com/appscode/voyager/pull/877) ([tamalsaha](https://github.com/tamalsaha))
- Update docs for supported annotations [\#871](https://github.com/appscode/voyager/pull/871) ([diptadas](https://github.com/diptadas))

## [6.0.0-rc.0](https://github.com/appscode/voyager/tree/6.0.0-rc.0) (2018-02-14)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.11...6.0.0-rc.0)

**Fixed bugs:**

- Document GKE cluster RBAC setup [\#564](https://github.com/appscode/voyager/issues/564)

**Closed issues:**

- LoadBalancer vs NodePort with manualy setup LB \(haproxy.cfg difference\) [\#867](https://github.com/appscode/voyager/issues/867)
- Ignore Rule if backend service is missing [\#848](https://github.com/appscode/voyager/issues/848)
- Failed to list ServiceMonitor [\#847](https://github.com/appscode/voyager/issues/847)
- Uninstall deletes object only from kube-system namespace [\#846](https://github.com/appscode/voyager/issues/846)
- Multi backends for one domain [\#833](https://github.com/appscode/voyager/issues/833)
- TCP Ingress Health Check Annotations not Working [\#832](https://github.com/appscode/voyager/issues/832)
- DNS-01 Challenge provider missing key in credential [\#821](https://github.com/appscode/voyager/issues/821)
- Allow users to specify backend names [\#819](https://github.com/appscode/voyager/issues/819)
- Should we make acl names part of the "api"? [\#818](https://github.com/appscode/voyager/issues/818)
- ACL generation: Support cookie matching [\#817](https://github.com/appscode/voyager/issues/817)
- Default http-\>https redirect turns DELETE \(and possibly other HTTP verbs\) into GET [\#816](https://github.com/appscode/voyager/issues/816)
- Panic in runtime.go when using TLS [\#814](https://github.com/appscode/voyager/issues/814)
- ACL generation: Support multiple path matching per rule [\#813](https://github.com/appscode/voyager/issues/813)
- ACL in haproxy not created correctly when an ingress has a single host rule [\#807](https://github.com/appscode/voyager/issues/807)
- Constant "Back-off restarting failed container" for a non existing bad ingress. [\#797](https://github.com/appscode/voyager/issues/797)
- When a pod linked to a service is deleted, Voyager Operator crashes and does not update ConfigMap [\#790](https://github.com/appscode/voyager/issues/790)
- Pod reboot loop with "One or more Ingress objects are invalid" [\#779](https://github.com/appscode/voyager/issues/779)
- Using Voyager and Let's Encrypt in multiple Kubernetes clusters in different regions [\#687](https://github.com/appscode/voyager/issues/687)
- Self-referential Ingress and Certificate must be done in order [\#661](https://github.com/appscode/voyager/issues/661)
- GRPC example [\#604](https://github.com/appscode/voyager/issues/604)
- Websocket example [\#603](https://github.com/appscode/voyager/issues/603)
- Support direct scrapping via Prometheus [\#593](https://github.com/appscode/voyager/issues/593)
- Use field selectors in TLS mounters [\#558](https://github.com/appscode/voyager/issues/558)
- Update Voyager to use workqueue [\#535](https://github.com/appscode/voyager/issues/535)
- Test 3.2.0 to 5.0.0 migration is smooth [\#527](https://github.com/appscode/voyager/issues/527)
- Change BackendRule to BackendRules [\#468](https://github.com/appscode/voyager/issues/468)
- Use Kutil based PATCH to apply changes [\#457](https://github.com/appscode/voyager/issues/457)
- Use Secret to store HAProxy.conf [\#447](https://github.com/appscode/voyager/issues/447)
- voyager check should check annotations and dump the parsed annotations [\#367](https://github.com/appscode/voyager/issues/367)
- Document IAM permission needed for HostPort mode [\#358](https://github.com/appscode/voyager/issues/358)
- Canonicalize TemplateData [\#348](https://github.com/appscode/voyager/issues/348)

**Merged pull requests:**

- Remove bad acl from haproxy template [\#875](https://github.com/appscode/voyager/pull/875) ([tamalsaha](https://github.com/tamalsaha))
- annotations.md typo fix [\#874](https://github.com/appscode/voyager/pull/874) ([mu5h3r](https://github.com/mu5h3r))
- Use service port by default for LB type nodeport [\#870](https://github.com/appscode/voyager/pull/870) ([diptadas](https://github.com/diptadas))
- Fixed configmap cleanup when ingress deleted [\#869](https://github.com/appscode/voyager/pull/869) ([diptadas](https://github.com/diptadas))
- Removed deprecated sticky annotation [\#868](https://github.com/appscode/voyager/pull/868) ([diptadas](https://github.com/diptadas))
- Pass client config to webhook [\#865](https://github.com/appscode/voyager/pull/865) ([tamalsaha](https://github.com/tamalsaha))
- Fixed e2e tests [\#863](https://github.com/appscode/voyager/pull/863) ([diptadas](https://github.com/diptadas))
- Update charts to support api registration [\#862](https://github.com/appscode/voyager/pull/862) ([tamalsaha](https://github.com/tamalsaha))
- Use ${} form for onessl envsubst [\#861](https://github.com/appscode/voyager/pull/861) ([tamalsaha](https://github.com/tamalsaha))
- Ignore error for missing backend services [\#860](https://github.com/appscode/voyager/pull/860) ([diptadas](https://github.com/diptadas))
- Make operator run locally [\#859](https://github.com/appscode/voyager/pull/859) ([tamalsaha](https://github.com/tamalsaha))
- Update comment regarding RBAC [\#858](https://github.com/appscode/voyager/pull/858) ([bcyrill](https://github.com/bcyrill))
- Don't append duplicate group versions [\#857](https://github.com/appscode/voyager/pull/857) ([tamalsaha](https://github.com/tamalsaha))
- Merge admission webhook and operator into one binary [\#856](https://github.com/appscode/voyager/pull/856) ([tamalsaha](https://github.com/tamalsaha))
- Install admission webhook for Kubernetes \>=1.9.0 [\#855](https://github.com/appscode/voyager/pull/855) ([tamalsaha](https://github.com/tamalsaha))
- Merge uninstall script into the voyager.sh script [\#854](https://github.com/appscode/voyager/pull/854) ([tamalsaha](https://github.com/tamalsaha))
- Fixed panic during annotation parsing [\#853](https://github.com/appscode/voyager/pull/853) ([diptadas](https://github.com/diptadas))
- Checked timeout and dns-resolver maps [\#852](https://github.com/appscode/voyager/pull/852) ([diptadas](https://github.com/diptadas))
- Add missing RBAC for ServiceMonitor [\#851](https://github.com/appscode/voyager/pull/851) ([tamalsaha](https://github.com/tamalsaha))
- Document GKE permission options [\#850](https://github.com/appscode/voyager/pull/850) ([tamalsaha](https://github.com/tamalsaha))
- Ignore --run-on-master flags for GKE [\#849](https://github.com/appscode/voyager/pull/849) ([tamalsaha](https://github.com/tamalsaha))
- Change BackendRule to BackendRules [\#845](https://github.com/appscode/voyager/pull/845) ([tamalsaha](https://github.com/tamalsaha))
- Type check for annotations in validator  [\#844](https://github.com/appscode/voyager/pull/844) ([diptadas](https://github.com/diptadas))
- Revise host and path acl names to make them part of "api" [\#843](https://github.com/appscode/voyager/pull/843) ([tamalsaha](https://github.com/tamalsaha))
- Preserve original HTTP verb on redirect [\#842](https://github.com/appscode/voyager/pull/842) ([tamalsaha](https://github.com/tamalsaha))
- Only assign deployment replicas initially [\#841](https://github.com/appscode/voyager/pull/841) ([diptadas](https://github.com/diptadas))
- Fix DNS-01 Challenge provider missing key in credential [\#840](https://github.com/appscode/voyager/pull/840) ([tamalsaha](https://github.com/tamalsaha))
- Checked for invalid backend service name in validator [\#839](https://github.com/appscode/voyager/pull/839) ([diptadas](https://github.com/diptadas))
- Removed panic in operator for bad-ingress [\#837](https://github.com/appscode/voyager/pull/837) ([diptadas](https://github.com/diptadas))
- Checked nil backend before assigning [\#836](https://github.com/appscode/voyager/pull/836) ([diptadas](https://github.com/diptadas))
- Copy generic-admission-server code into pkg [\#835](https://github.com/appscode/voyager/pull/835) ([tamalsaha](https://github.com/tamalsaha))
- Log TemplateData in debug mode [\#834](https://github.com/appscode/voyager/pull/834) ([tamalsaha](https://github.com/tamalsaha))
-  Removed maps from template data [\#831](https://github.com/appscode/voyager/pull/831) ([diptadas](https://github.com/diptadas))
- Prepare docs for 6.0.0-alpha.0 [\#830](https://github.com/appscode/voyager/pull/830) ([tamalsaha](https://github.com/tamalsaha))
- Support private docker registry in installer [\#829](https://github.com/appscode/voyager/pull/829) ([tamalsaha](https://github.com/tamalsaha))
- Add ValidatingAdmissionWebhook for Voyager CRDs [\#828](https://github.com/appscode/voyager/pull/828) ([tamalsaha](https://github.com/tamalsaha))
- Use kubectl auth reconcile in installer script [\#827](https://github.com/appscode/voyager/pull/827) ([tamalsaha](https://github.com/tamalsaha))
- Update changelog [\#826](https://github.com/appscode/voyager/pull/826) ([tamalsaha](https://github.com/tamalsaha))
- Update client-go to 6.0.0 [\#825](https://github.com/appscode/voyager/pull/825) ([tamalsaha](https://github.com/tamalsaha))
- Update copyright year to 2018 [\#824](https://github.com/appscode/voyager/pull/824) ([tamalsaha](https://github.com/tamalsaha))
- Merge tls-mounter & kloader into haproxy-controller [\#823](https://github.com/appscode/voyager/pull/823) ([tamalsaha](https://github.com/tamalsaha))
- Updating kube-mon so service-monitor-endpoint-port is optional [\#822](https://github.com/appscode/voyager/pull/822) ([jeffersongirao](https://github.com/jeffersongirao))
- Fix unit tests [\#820](https://github.com/appscode/voyager/pull/820) ([jeffersongirao](https://github.com/jeffersongirao))
- Use deterministic-suffix instead of random-suffix in backend name [\#815](https://github.com/appscode/voyager/pull/815) ([diptadas](https://github.com/diptadas))
- Ignored not-found error for DNS resolver annotations  [\#812](https://github.com/appscode/voyager/pull/812) ([diptadas](https://github.com/diptadas))
- Add prometheus flags to command that uses it [\#810](https://github.com/appscode/voyager/pull/810) ([tamalsaha](https://github.com/tamalsaha))
- Improve concepts docs [\#809](https://github.com/appscode/voyager/pull/809) ([tamalsaha](https://github.com/tamalsaha))
- Revendor coreos prometheus operator 0.16.0 [\#808](https://github.com/appscode/voyager/pull/808) ([tamalsaha](https://github.com/tamalsaha))
- Revendor log wrapper [\#804](https://github.com/appscode/voyager/pull/804) ([tamalsaha](https://github.com/tamalsaha))
- Implement work-queue in operator [\#803](https://github.com/appscode/voyager/pull/803) ([diptadas](https://github.com/diptadas))
- Fix links in chart [\#802](https://github.com/appscode/voyager/pull/802) ([tamalsaha](https://github.com/tamalsaha))
- Add changelog [\#801](https://github.com/appscode/voyager/pull/801) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.11](https://github.com/appscode/voyager/tree/5.0.0-rc.11) (2018-01-04)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.10...5.0.0-rc.11)

**Fixed bugs:**

- Avoid unnecessary config reloads in HAProxy [\#512](https://github.com/appscode/voyager/issues/512)
- Allow adding new domain to cert crd [\#788](https://github.com/appscode/voyager/pull/788) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Support all annotations under ingress.appscode.com key [\#791](https://github.com/appscode/voyager/issues/791)
- expose port on host  [\#778](https://github.com/appscode/voyager/issues/778)
- Missing Ingress Annotation in Documentation [\#668](https://github.com/appscode/voyager/issues/668)
- Support additional CORS headers [\#656](https://github.com/appscode/voyager/issues/656)

**Merged pull requests:**

- Prepare docs for 5.0.0-rc.11 [\#799](https://github.com/appscode/voyager/pull/799) ([tamalsaha](https://github.com/tamalsaha))
- Reorganize docs for hosting on product site [\#798](https://github.com/appscode/voyager/pull/798) ([tamalsaha](https://github.com/tamalsaha))
- Detect client id from ENV [\#795](https://github.com/appscode/voyager/pull/795) ([tamalsaha](https://github.com/tamalsaha))
- Update dead links [\#794](https://github.com/appscode/voyager/pull/794) ([petedreizehn](https://github.com/petedreizehn))
- Support additional CORS headers [\#793](https://github.com/appscode/voyager/pull/793) ([diptadas](https://github.com/diptadas))
- Support ingress.appscode.com key for all annotations [\#792](https://github.com/appscode/voyager/pull/792) ([diptadas](https://github.com/diptadas))
- Use CertStore from kutil [\#789](https://github.com/appscode/voyager/pull/789) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.10](https://github.com/appscode/voyager/tree/5.0.0-rc.10) (2017-12-29)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.9...5.0.0-rc.10)

**Fixed bugs:**

- Set selector for headless service of a HostPort ingress [\#785](https://github.com/appscode/voyager/pull/785) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Issues with ACME well-known paths [\#787](https://github.com/appscode/voyager/issues/787)

**Merged pull requests:**

- Generate host acl correctly for `\*` host [\#786](https://github.com/appscode/voyager/pull/786) ([tamalsaha](https://github.com/tamalsaha))
- Add front matter for docs 5.0.0-rc.9 [\#784](https://github.com/appscode/voyager/pull/784) ([sajibcse68](https://github.com/sajibcse68))

## [5.0.0-rc.9](https://github.com/appscode/voyager/tree/5.0.0-rc.9) (2017-12-28)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.8...5.0.0-rc.9)

**Fixed bugs:**

- Move Acme paths to top of path list [\#781](https://github.com/appscode/voyager/pull/781) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Baremetal setup not working at all [\#780](https://github.com/appscode/voyager/issues/780)
- Patching voyager ingress fails [\#773](https://github.com/appscode/voyager/issues/773)

**Merged pull requests:**

- Prepare docs for 5.0.0-rc.9 [\#782](https://github.com/appscode/voyager/pull/782) ([tamalsaha](https://github.com/tamalsaha))
- Use cmp methods from kutil [\#777](https://github.com/appscode/voyager/pull/777) ([tamalsaha](https://github.com/tamalsaha))
- Show how to run haproxy pods on master [\#776](https://github.com/appscode/voyager/pull/776) ([tamalsaha](https://github.com/tamalsaha))
- Use verb type to indicate mutation [\#775](https://github.com/appscode/voyager/pull/775) ([tamalsaha](https://github.com/tamalsaha))
- Use kube-mon repo [\#774](https://github.com/appscode/voyager/pull/774) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.8](https://github.com/appscode/voyager/tree/5.0.0-rc.8) (2017-12-20)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.7...5.0.0-rc.8)

**Fixed bugs:**

- Fix backend name checking for haproxy template [\#771](https://github.com/appscode/voyager/pull/771) ([tamalsaha](https://github.com/tamalsaha))
- Fix installation instructions in guides [\#770](https://github.com/appscode/voyager/pull/770) ([tamalsaha](https://github.com/tamalsaha))
- Support wildcard in TLS searching [\#768](https://github.com/appscode/voyager/pull/768) ([tamalsaha](https://github.com/tamalsaha))
- Merge monitor service ports correctly [\#767](https://github.com/appscode/voyager/pull/767) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Update docs for 5.0.0-rc.8 [\#772](https://github.com/appscode/voyager/pull/772) ([tamalsaha](https://github.com/tamalsaha))
- Document how to use external-ip [\#769](https://github.com/appscode/voyager/pull/769) ([tamalsaha](https://github.com/tamalsaha))
- Update RBAC for analytics [\#766](https://github.com/appscode/voyager/pull/766) ([tamalsaha](https://github.com/tamalsaha))
- Set ClientID for analytics [\#765](https://github.com/appscode/voyager/pull/765) ([tamalsaha](https://github.com/tamalsaha))
- Rename tasks to guides [\#764](https://github.com/appscode/voyager/pull/764) ([tamalsaha](https://github.com/tamalsaha))
- Revise ingress docs [\#755](https://github.com/appscode/voyager/pull/755) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.7](https://github.com/appscode/voyager/tree/5.0.0-rc.7) (2017-12-13)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.6...5.0.0-rc.7)

**Closed issues:**

- List of created ingresses? [\#745](https://github.com/appscode/voyager/issues/745)
- create san cert with panic [\#744](https://github.com/appscode/voyager/issues/744)
- \[Feature request\] Support for tolerations in ingress pod spec [\#503](https://github.com/appscode/voyager/issues/503)
- Document AWS IAM permissions for LE DNS validation. [\#337](https://github.com/appscode/voyager/issues/337)

**Merged pull requests:**

- Prepare for 5.0.0-rc.7 release [\#757](https://github.com/appscode/voyager/pull/757) ([tamalsaha](https://github.com/tamalsaha))
- Installer for custom template [\#756](https://github.com/appscode/voyager/pull/756) ([tamalsaha](https://github.com/tamalsaha))
- Change left\_menu -\> menu\_name [\#748](https://github.com/appscode/voyager/pull/748) ([tamalsaha](https://github.com/tamalsaha))
- Fix panic when crt.status.LastIssuedCertificate is missing on renew [\#746](https://github.com/appscode/voyager/pull/746) ([tamalsaha](https://github.com/tamalsaha))
- Use RegisterCRDs from kutil [\#743](https://github.com/appscode/voyager/pull/743) ([tamalsaha](https://github.com/tamalsaha))
- Document updated cert manager [\#581](https://github.com/appscode/voyager/pull/581) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.6](https://github.com/appscode/voyager/tree/5.0.0-rc.6) (2017-12-05)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.5...5.0.0-rc.6)

**Merged pull requests:**

- Use forked golang/x/oauth2 library [\#741](https://github.com/appscode/voyager/pull/741) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 5.0.0-rc.6 release [\#739](https://github.com/appscode/voyager/pull/739) ([tamalsaha](https://github.com/tamalsaha))
- Avoid duplicate ACLs for host [\#738](https://github.com/appscode/voyager/pull/738) ([tamalsaha](https://github.com/tamalsaha))
- Trim space from ACME user email [\#737](https://github.com/appscode/voyager/pull/737) ([tamalsaha](https://github.com/tamalsaha))
- Revendor dependencies [\#736](https://github.com/appscode/voyager/pull/736) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.5](https://github.com/appscode/voyager/tree/5.0.0-rc.5) (2017-12-01)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.4...5.0.0-rc.5)

**Fixed bugs:**

- No certificates were found while parsing the bundle. [\#725](https://github.com/appscode/voyager/issues/725)

**Merged pull requests:**

- Prepare docs for 5.0.0-rc.5 release [\#735](https://github.com/appscode/voyager/pull/735) ([tamalsaha](https://github.com/tamalsaha))
- Correctly encode cert for renewal. [\#734](https://github.com/appscode/voyager/pull/734) ([tamalsaha](https://github.com/tamalsaha))
- Add aliases for README file [\#731](https://github.com/appscode/voyager/pull/731) ([sajibcse68](https://github.com/sajibcse68))
- Update version in front matter for docs [\#730](https://github.com/appscode/voyager/pull/730) ([tamalsaha](https://github.com/tamalsaha))
- Add Docs Front Matter [\#728](https://github.com/appscode/voyager/pull/728) ([sajibcse68](https://github.com/sajibcse68))

## [5.0.0-rc.4](https://github.com/appscode/voyager/tree/5.0.0-rc.4) (2017-11-28)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.3...5.0.0-rc.4)

**Implemented enhancements:**

- Print namespace of missing service name [\#710](https://github.com/appscode/voyager/issues/710)
- Support Health Check for backend [\#683](https://github.com/appscode/voyager/issues/683)
- Allow send-ing proxy header to backend [\#164](https://github.com/appscode/voyager/issues/164)

**Fixed bugs:**

- Check voyager respects --ingress-class flag [\#711](https://github.com/appscode/voyager/issues/711)
- Adding annotation `ingress.kubernetes.io/hsts` makes voyager generate invalid haproxy config [\#701](https://github.com/appscode/voyager/issues/701)
- Perform ssl-redirect after matching host [\#691](https://github.com/appscode/voyager/issues/691)
- haproxy.cfg:42 rsprep error [\#678](https://github.com/appscode/voyager/issues/678)
- Fix ssl-passthrough [\#665](https://github.com/appscode/voyager/issues/665)
- HTTP -\> HTTPS redirection does not work in 1.8 cluster with AWS cert manager [\#639](https://github.com/appscode/voyager/issues/639)
- Don't use backend name to generate acl name [\#726](https://github.com/appscode/voyager/pull/726) ([tamalsaha](https://github.com/tamalsaha))
- Unconditionally set headers defined in Ingress [\#717](https://github.com/appscode/voyager/pull/717) ([tamalsaha](https://github.com/tamalsaha))
- Correctly handle updated ingress.class annotation [\#715](https://github.com/appscode/voyager/pull/715) ([tamalsaha](https://github.com/tamalsaha))
- Support aws or route53 as providers which read dns credential from ENV [\#712](https://github.com/appscode/voyager/pull/712) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Stop cross namespace support when restricted to one namespace [\#698](https://github.com/appscode/voyager/issues/698)
- One or more Ingress objects are invalid [\#697](https://github.com/appscode/voyager/issues/697)
- Cannot create TCP ingress in k8s 1.8.2 and voyager 5.0.0-rc3 [\#696](https://github.com/appscode/voyager/issues/696)
- monitor openstack [\#694](https://github.com/appscode/voyager/issues/694)
- Configure HAProxy to terminate SSL and send PROXYv2 [\#692](https://github.com/appscode/voyager/issues/692)
- voyager 5-rc3 non kube-system ingress , error [\#689](https://github.com/appscode/voyager/issues/689)
- error creating a very simple object on version 5-rc3 [\#688](https://github.com/appscode/voyager/issues/688)
- Support ExternalIPs [\#686](https://github.com/appscode/voyager/issues/686)
- Support rewrite-target annotation [\#657](https://github.com/appscode/voyager/issues/657)
- Document importance to order of paths [\#422](https://github.com/appscode/voyager/issues/422)

**Merged pull requests:**

- Cleanup wildcard in ACL name [\#727](https://github.com/appscode/voyager/pull/727) ([tamalsaha](https://github.com/tamalsaha))
- Load AWS\_HOSTED\_ZONE\_ID if provided by user [\#724](https://github.com/appscode/voyager/pull/724) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 5.0.0-rc.4 [\#723](https://github.com/appscode/voyager/pull/723) ([tamalsaha](https://github.com/tamalsaha))
- Make voyager YAMLs installable from internet [\#722](https://github.com/appscode/voyager/pull/722) ([tamalsaha](https://github.com/tamalsaha))
- Add front matter for voyager cli ref [\#721](https://github.com/appscode/voyager/pull/721) ([tamalsaha](https://github.com/tamalsaha))
- Support rewrite-target annotation [\#720](https://github.com/appscode/voyager/pull/720) ([tamalsaha](https://github.com/tamalsaha))
- Print namespace of missing service name [\#716](https://github.com/appscode/voyager/pull/716) ([tamalsaha](https://github.com/tamalsaha))
- Use http-response set-header instead of rspadd [\#714](https://github.com/appscode/voyager/pull/714) ([tamalsaha](https://github.com/tamalsaha))
- Use const for test domain [\#713](https://github.com/appscode/voyager/pull/713) ([tamalsaha](https://github.com/tamalsaha))
- Don't allow cross ns backend when voyager is restricted to own ns [\#709](https://github.com/appscode/voyager/pull/709) ([tamalsaha](https://github.com/tamalsaha))
- Document azure support for load-balancer-ip [\#708](https://github.com/appscode/voyager/pull/708) ([tamalsaha](https://github.com/tamalsaha))
- Convert rules for SSL Passthrough [\#706](https://github.com/appscode/voyager/pull/706) ([diptadas](https://github.com/diptadas))
- Keep all newlines in haproxy.cfg [\#705](https://github.com/appscode/voyager/pull/705) ([tamalsaha](https://github.com/tamalsaha))
- Revise StatsAccessor interface [\#704](https://github.com/appscode/voyager/pull/704) ([tamalsaha](https://github.com/tamalsaha))
- Support direct scrapping via Prometheus [\#703](https://github.com/appscode/voyager/pull/703) ([tamalsaha](https://github.com/tamalsaha))
- Perform ssl-redirect after matching host [\#702](https://github.com/appscode/voyager/pull/702) ([tamalsaha](https://github.com/tamalsaha))
- Fix build [\#700](https://github.com/appscode/voyager/pull/700) ([tamalsaha](https://github.com/tamalsaha))
- Support PROXY protocol in test server [\#699](https://github.com/appscode/voyager/pull/699) ([diptadas](https://github.com/diptadas))
- Enable server health check using service annotations and backend rules [\#695](https://github.com/appscode/voyager/pull/695) ([diptadas](https://github.com/diptadas))
- Add to backends the options for send-proxy variants for server. [\#693](https://github.com/appscode/voyager/pull/693) ([drf](https://github.com/drf))
- Support ExternalIPs [\#690](https://github.com/appscode/voyager/pull/690) ([tamalsaha](https://github.com/tamalsaha))
- Use DeepCopy with PATCH calls. [\#685](https://github.com/appscode/voyager/pull/685) ([tamalsaha](https://github.com/tamalsaha))
- Fix template rendering [\#682](https://github.com/appscode/voyager/pull/682) ([tamalsaha](https://github.com/tamalsaha))
- Move chart inside stable folder [\#681](https://github.com/appscode/voyager/pull/681) ([tamalsaha](https://github.com/tamalsaha))
- Make chart namespaced [\#680](https://github.com/appscode/voyager/pull/680) ([tamalsaha](https://github.com/tamalsaha))
- Allow for binding HTTP or TCP ingress rules to specific addresses [\#649](https://github.com/appscode/voyager/pull/649) ([deuill](https://github.com/deuill))

## [5.0.0-rc.3](https://github.com/appscode/voyager/tree/5.0.0-rc.3) (2017-11-02)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.2...5.0.0-rc.3)

**Closed issues:**

- Support imagePullSecrets for HAProxy pods [\#673](https://github.com/appscode/voyager/issues/673)
- Document how to configure DNS in Hostport / NodePort mode [\#354](https://github.com/appscode/voyager/issues/354)

**Merged pull requests:**

- Add  image/tag variables in chart [\#677](https://github.com/appscode/voyager/pull/677) ([tamalsaha](https://github.com/tamalsaha))
- Detect change in imagePullSecrets [\#676](https://github.com/appscode/voyager/pull/676) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 5.0.0-rc.3 [\#675](https://github.com/appscode/voyager/pull/675) ([tamalsaha](https://github.com/tamalsaha))
- Add ImagePullSecrets in Ingress [\#674](https://github.com/appscode/voyager/pull/674) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.2](https://github.com/appscode/voyager/tree/5.0.0-rc.2) (2017-11-02)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.1...5.0.0-rc.2)

**Fixed bugs:**

- Add `deletecollection` permission to voyager operator [\#666](https://github.com/appscode/voyager/pull/666) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Support GoDaddy DNS provider [\#672](https://github.com/appscode/voyager/pull/672) ([tamalsaha](https://github.com/tamalsaha))
- Support openstack provider [\#671](https://github.com/appscode/voyager/pull/671) ([tamalsaha](https://github.com/tamalsaha))
- Support `ingress.appscode.com/keep-source-ip` annotation for NodePort mode [\#667](https://github.com/appscode/voyager/pull/667) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.1](https://github.com/appscode/voyager/tree/5.0.0-rc.1) (2017-10-26)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-alpha.1...5.0.0-rc.1)

**Fixed bugs:**

- TCP mode does not work in port 80 [\#663](https://github.com/appscode/voyager/issues/663)

**Merged pull requests:**

- Enable TCP mode in port 80 [\#664](https://github.com/appscode/voyager/pull/664) ([tamalsaha](https://github.com/tamalsaha))
- Remove unused fields from LocalTypedReference [\#662](https://github.com/appscode/voyager/pull/662) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-alpha.1](https://github.com/appscode/voyager/tree/5.0.0-alpha.1) (2017-10-24)
[Full Changelog](https://github.com/appscode/voyager/compare/5.0.0-rc.0...5.0.0-alpha.1)

**Fixed bugs:**

- Avoid redirecting ACME requests to https scheme [\#660](https://github.com/appscode/voyager/pull/660) ([tamalsaha](https://github.com/tamalsaha))

## [5.0.0-rc.0](https://github.com/appscode/voyager/tree/5.0.0-rc.0) (2017-10-23)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.18...5.0.0-rc.0)

**Implemented enhancements:**

- Allow for binding frontends to specific addresses [\#602](https://github.com/appscode/voyager/issues/602)

**Fixed bugs:**

- Fix Certificate Test Name [\#648](https://github.com/appscode/voyager/pull/648) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Use typed versioned client for CRD [\#659](https://github.com/appscode/voyager/pull/659) ([tamalsaha](https://github.com/tamalsaha))
- Use prometheus-operator v1 api/client [\#658](https://github.com/appscode/voyager/pull/658) ([tamalsaha](https://github.com/tamalsaha))
- Fix project name in header for auto generated files [\#655](https://github.com/appscode/voyager/pull/655) ([tamalsaha](https://github.com/tamalsaha))
- Document the important of order of paths [\#654](https://github.com/appscode/voyager/pull/654) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 5.0.0-rc.0 [\#653](https://github.com/appscode/voyager/pull/653) ([tamalsaha](https://github.com/tamalsaha))
- Update prometheus-operator to implement DeepCopy\(\) [\#652](https://github.com/appscode/voyager/pull/652) ([tamalsaha](https://github.com/tamalsaha))
- Fix NPE in time.Equal method [\#651](https://github.com/appscode/voyager/pull/651) ([tamalsaha](https://github.com/tamalsaha))
- Change `k8s.io/api/core/v1` pkg alias to core [\#650](https://github.com/appscode/voyager/pull/650) ([tamalsaha](https://github.com/tamalsaha))
- Update developer-guide [\#642](https://github.com/appscode/voyager/pull/642) ([sadlil](https://github.com/sadlil))
- Use client-go 5.x [\#629](https://github.com/appscode/voyager/pull/629) ([tamalsaha](https://github.com/tamalsaha))
- Generate openapi spec [\#596](https://github.com/appscode/voyager/pull/596) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.18](https://github.com/appscode/voyager/tree/4.0.0-rc.18) (2017-10-18)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.17...4.0.0-rc.18)

**Fixed bugs:**

- Make AWS\_ACCESS\_KEY\_ID optional [\#644](https://github.com/appscode/voyager/pull/644) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Operator doesn't create CRD groups [\#643](https://github.com/appscode/voyager/issues/643)

## [4.0.0-rc.17](https://github.com/appscode/voyager/tree/4.0.0-rc.17) (2017-10-18)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.16...4.0.0-rc.17)

**Fixed bugs:**

- Test wildcard domains work with TLS [\#598](https://github.com/appscode/voyager/issues/598)

**Closed issues:**

- Up kubernetes/client-go QPS and Burst config [\#640](https://github.com/appscode/voyager/issues/640)

**Merged pull requests:**

- Raise kubernetes/client-go QPS and Burst config [\#641](https://github.com/appscode/voyager/pull/641) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.16](https://github.com/appscode/voyager/tree/4.0.0-rc.16) (2017-10-16)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.15...4.0.0-rc.16)

**Fixed bugs:**

- haproxy points to wrong file on tcp+tls config [\#630](https://github.com/appscode/voyager/issues/630)

**Closed issues:**

- Support `ingress.appscode.com/type: internal` [\#627](https://github.com/appscode/voyager/issues/627)

**Merged pull requests:**

- Implement `ingress.appscode.com/type: internal`  [\#636](https://github.com/appscode/voyager/pull/636) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.15](https://github.com/appscode/voyager/tree/4.0.0-rc.15) (2017-10-16)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.14...4.0.0-rc.15)

**Fixed bugs:**

- Fix tcp frontend template [\#634](https://github.com/appscode/voyager/pull/634) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Update chart helper truncate length [\#633](https://github.com/appscode/voyager/pull/633) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.14](https://github.com/appscode/voyager/tree/4.0.0-rc.14) (2017-10-16)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.13...4.0.0-rc.14)

**Fixed bugs:**

- All Tests and Bug fixes for release-4 [\#628](https://github.com/appscode/voyager/pull/628) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Rename SecretName to CertFile [\#632](https://github.com/appscode/voyager/pull/632) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.13](https://github.com/appscode/voyager/tree/4.0.0-rc.13) (2017-10-16)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.12...4.0.0-rc.13)

**Fixed bugs:**

- Replace reflect.Equal with github.com/google/go-cmp [\#626](https://github.com/appscode/voyager/pull/626) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Update unit tests [\#623](https://github.com/appscode/voyager/pull/623) ([julianvmodesto](https://github.com/julianvmodesto))

## [4.0.0-rc.12](https://github.com/appscode/voyager/tree/4.0.0-rc.12) (2017-10-13)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.11...4.0.0-rc.12)

**Closed issues:**

- Add Voyager to official ingress project docs. [\#437](https://github.com/appscode/voyager/issues/437)

**Merged pull requests:**

- Prepare docs for 4.0.0-rc.12 [\#622](https://github.com/appscode/voyager/pull/622) ([tamalsaha](https://github.com/tamalsaha))
- Support TLS auth annotations [\#621](https://github.com/appscode/voyager/pull/621) ([tamalsaha](https://github.com/tamalsaha))
- Support Basic auth in FrontendRules [\#617](https://github.com/appscode/voyager/pull/617) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.11](https://github.com/appscode/voyager/tree/4.0.0-rc.11) (2017-10-12)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.10...4.0.0-rc.11)

**Implemented enhancements:**

- TLS auth [\#606](https://github.com/appscode/voyager/pull/606) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Upgrade Prometheus Operator [\#608](https://github.com/appscode/voyager/issues/608)
- Watch secrets and update the config when Basic auth changes [\#560](https://github.com/appscode/voyager/issues/560)
- TLS auth [\#606](https://github.com/appscode/voyager/pull/606) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Allow restricting voyager in a single namespace [\#582](https://github.com/appscode/voyager/issues/582)
- zone-specific static IP on gke rather than global static [\#414](https://github.com/appscode/voyager/issues/414)
- Add flag to handling standard ingress [\#369](https://github.com/appscode/voyager/issues/369)

**Merged pull requests:**

- Allow restricting voyager in a single namespace [\#619](https://github.com/appscode/voyager/pull/619) ([tamalsaha](https://github.com/tamalsaha))
- Add support for CRL when using TLS Auth [\#618](https://github.com/appscode/voyager/pull/618) ([tamalsaha](https://github.com/tamalsaha))
- Support ingress.kubernetes.io/ssl-redirect [\#616](https://github.com/appscode/voyager/pull/616) ([tamalsaha](https://github.com/tamalsaha))
- Remove support for ingress.appscode.com/egress-points annotations [\#615](https://github.com/appscode/voyager/pull/615) ([tamalsaha](https://github.com/tamalsaha))
- Add Wildcard domain Test [\#614](https://github.com/appscode/voyager/pull/614) ([sadlil](https://github.com/sadlil))
- Move CRD definition to api folder. [\#613](https://github.com/appscode/voyager/pull/613) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.10](https://github.com/appscode/voyager/tree/4.0.0-rc.10) (2017-10-10)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.9...4.0.0-rc.10)

**Fixed bugs:**

- Don't reload HAProxy using tls mounter setup phase [\#610](https://github.com/appscode/voyager/pull/610) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Change test domain appscode.dev -\> appscode.test [\#590](https://github.com/appscode/voyager/issues/590)

**Merged pull requests:**

- Clarify Prometheus operator version [\#612](https://github.com/appscode/voyager/pull/612) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 4.0.0-rc.10 release [\#611](https://github.com/appscode/voyager/pull/611) ([tamalsaha](https://github.com/tamalsaha))
- Update Prometheus operator dependency to 0.13.0 [\#609](https://github.com/appscode/voyager/pull/609) ([tamalsaha](https://github.com/tamalsaha))
- Add doc showing how to detect operator version [\#607](https://github.com/appscode/voyager/pull/607) ([tamalsaha](https://github.com/tamalsaha))
- Secret Update reflection [\#605](https://github.com/appscode/voyager/pull/605) ([sadlil](https://github.com/sadlil))
- Use .test TLD [\#601](https://github.com/appscode/voyager/pull/601) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.9](https://github.com/appscode/voyager/tree/4.0.0-rc.9) (2017-10-08)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.8...4.0.0-rc.9)

**Implemented enhancements:**

- Limit Connections [\#571](https://github.com/appscode/voyager/pull/571) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Fix validator so can specify either HTTP or TCP [\#597](https://github.com/appscode/voyager/pull/597) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Enable stats for e2e test [\#595](https://github.com/appscode/voyager/pull/595) ([tamalsaha](https://github.com/tamalsaha))
- Fix stats auth indentation when auth is omitted [\#594](https://github.com/appscode/voyager/pull/594) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.8](https://github.com/appscode/voyager/tree/4.0.0-rc.8) (2017-10-06)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.7...4.0.0-rc.8)

**Fixed bugs:**

- Assume cert store as Secret, if Vault missing. [\#592](https://github.com/appscode/voyager/pull/592) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.7](https://github.com/appscode/voyager/tree/4.0.0-rc.7) (2017-10-06)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.6...4.0.0-rc.7)

**Fixed bugs:**

- Migrate Ingress before projection [\#591](https://github.com/appscode/voyager/pull/591) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.6](https://github.com/appscode/voyager/tree/4.0.0-rc.6) (2017-10-06)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.5...4.0.0-rc.6)

**Fixed bugs:**

- LE: Too many invalid authorizations recently [\#587](https://github.com/appscode/voyager/issues/587)
- Fix HTTP challenger [\#589](https://github.com/appscode/voyager/pull/589) ([tamalsaha](https://github.com/tamalsaha))
- Inject well-known/acme-challenge path at the top of rules [\#588](https://github.com/appscode/voyager/pull/588) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.5](https://github.com/appscode/voyager/tree/4.0.0-rc.5) (2017-10-06)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.4...4.0.0-rc.5)

**Implemented enhancements:**

- Allow configuration of error files [\#525](https://github.com/appscode/voyager/issues/525)

**Fixed bugs:**

- Support static ip for Azure/ACS cluster. [\#584](https://github.com/appscode/voyager/pull/584) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- loadBalancerIP is ignored in azure mode [\#572](https://github.com/appscode/voyager/issues/572)
- Allow configuring options for each server entry [\#516](https://github.com/appscode/voyager/issues/516)
- Basic Auth annotations implementation [\#424](https://github.com/appscode/voyager/issues/424)

**Merged pull requests:**

- Prepare docs for 4.0.0-rc.5 [\#585](https://github.com/appscode/voyager/pull/585) ([tamalsaha](https://github.com/tamalsaha))
- Rename SecretRef to TLSRef [\#580](https://github.com/appscode/voyager/pull/580) ([tamalsaha](https://github.com/tamalsaha))
- Add LocalTypedReference type [\#579](https://github.com/appscode/voyager/pull/579) ([tamalsaha](https://github.com/tamalsaha))
- Add errofiles annotation [\#574](https://github.com/appscode/voyager/pull/574) ([diptadas](https://github.com/diptadas))
- Add force-ssl-redirect annotation [\#563](https://github.com/appscode/voyager/pull/563) ([diptadas](https://github.com/diptadas))

## [4.0.0-rc.4](https://github.com/appscode/voyager/tree/4.0.0-rc.4) (2017-10-05)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.2...4.0.0-rc.4)

**Fixed bugs:**

- Fix NodePort mode in GKE [\#575](https://github.com/appscode/voyager/pull/575) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Log GO's current thread id [\#573](https://github.com/appscode/voyager/issues/573)

**Merged pull requests:**

- Update docs for 4.0.0-rc.4 [\#576](https://github.com/appscode/voyager/pull/576) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.2](https://github.com/appscode/voyager/tree/3.2.2) (2017-10-05)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.3...3.2.2)

**Merged pull requests:**

- Disable OCSP must staple [\#570](https://github.com/appscode/voyager/pull/570) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.3](https://github.com/appscode/voyager/tree/4.0.0-rc.3) (2017-10-04)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.2...4.0.0-rc.3)

**Fixed bugs:**

- Add PATCH permission and fix deployment RBAC spec [\#568](https://github.com/appscode/voyager/pull/568) ([tamalsaha](https://github.com/tamalsaha))
- Fix RBAC permissions for apps/v1beta1 Deployments [\#565](https://github.com/appscode/voyager/pull/565) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Prepare docs for 4.0.0-rc.3 [\#569](https://github.com/appscode/voyager/pull/569) ([tamalsaha](https://github.com/tamalsaha))
- Set TypeMeta when creating object [\#567](https://github.com/appscode/voyager/pull/567) ([tamalsaha](https://github.com/tamalsaha))
- Fix logging [\#566](https://github.com/appscode/voyager/pull/566) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.2](https://github.com/appscode/voyager/tree/4.0.0-rc.2) (2017-10-04)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.1...4.0.0-rc.2)

**Fixed bugs:**

- Using HTTP challenge provider results in pod stuck at ContainerCreating stage [\#455](https://github.com/appscode/voyager/issues/455)

**Closed issues:**

- NodePort mode adds port to host header rule, but shouldn't [\#552](https://github.com/appscode/voyager/issues/552)
- Name server by pod name instead of endpoint ip [\#550](https://github.com/appscode/voyager/issues/550)
- ocsp stapling [\#531](https://github.com/appscode/voyager/issues/531)
- Bug: not creating RBAC roles in NodePort mode [\#524](https://github.com/appscode/voyager/issues/524)
- DNS resolver test is timing out [\#484](https://github.com/appscode/voyager/issues/484)

**Merged pull requests:**

- Prepare docs for 4.0.0-rc.2 [\#561](https://github.com/appscode/voyager/pull/561) ([tamalsaha](https://github.com/tamalsaha))
- Add ingress class support for helm chart [\#559](https://github.com/appscode/voyager/pull/559) ([xcompass](https://github.com/xcompass))
- Fix \#552 [\#557](https://github.com/appscode/voyager/pull/557) ([sadlil](https://github.com/sadlil))
- Docs for 4.0 - part 1 [\#556](https://github.com/appscode/voyager/pull/556) ([sadlil](https://github.com/sadlil))
- Add service auth annotation [\#555](https://github.com/appscode/voyager/pull/555) ([diptadas](https://github.com/diptadas))
- Don't log error if to-be-deleted object is missing. [\#554](https://github.com/appscode/voyager/pull/554) ([tamalsaha](https://github.com/tamalsaha))
- Generate ugorji stuff [\#553](https://github.com/appscode/voyager/pull/553) ([tamalsaha](https://github.com/tamalsaha))
-  Name server by pod name instead of endpoint ip [\#551](https://github.com/appscode/voyager/pull/551) ([sadlil](https://github.com/sadlil))
- Add max-connections annotation [\#546](https://github.com/appscode/voyager/pull/546) ([diptadas](https://github.com/diptadas))

## [4.0.0-rc.1](https://github.com/appscode/voyager/tree/4.0.0-rc.1) (2017-09-27)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-rc.0...4.0.0-rc.1)

**Fixed bugs:**

- Fix cert controller bugs [\#541](https://github.com/appscode/voyager/pull/541) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Install Voyager as critical addon [\#292](https://github.com/appscode/voyager/issues/292)

**Merged pull requests:**

- Fix test [\#549](https://github.com/appscode/voyager/pull/549) ([diptadas](https://github.com/diptadas))
- Add init-only mode for tls mounter [\#548](https://github.com/appscode/voyager/pull/548) ([tamalsaha](https://github.com/tamalsaha))
- Fix tls mounter [\#547](https://github.com/appscode/voyager/pull/547) ([sadlil](https://github.com/sadlil))
- Update docs to CRD from TPR [\#544](https://github.com/appscode/voyager/pull/544) ([tamalsaha](https://github.com/tamalsaha))
- Fix tls mounter [\#543](https://github.com/appscode/voyager/pull/543) ([tamalsaha](https://github.com/tamalsaha))
- Ensure RBAC if Ingress is updated [\#542](https://github.com/appscode/voyager/pull/542) ([tamalsaha](https://github.com/tamalsaha))
- Make SecretRef pointer again [\#540](https://github.com/appscode/voyager/pull/540) ([tamalsaha](https://github.com/tamalsaha))
- Add whitelist-source-range annotation [\#539](https://github.com/appscode/voyager/pull/539) ([diptadas](https://github.com/diptadas))
- Add links to user guide [\#537](https://github.com/appscode/voyager/pull/537) ([tamalsaha](https://github.com/tamalsaha))
- Install voyager operator as critical addon [\#536](https://github.com/appscode/voyager/pull/536) ([tamalsaha](https://github.com/tamalsaha))
- Remove UpdateRBAC mode. [\#534](https://github.com/appscode/voyager/pull/534) ([tamalsaha](https://github.com/tamalsaha))
- Use CreateOrPatch apis with RBAC. Also sets ownerReference. [\#533](https://github.com/appscode/voyager/pull/533) ([tamalsaha](https://github.com/tamalsaha))
- Disable OCSP must staple [\#532](https://github.com/appscode/voyager/pull/532) ([tamalsaha](https://github.com/tamalsaha))
- Explain why tcp connections can't be whitelisted for AWS LoadBlancers [\#514](https://github.com/appscode/voyager/pull/514) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-rc.0](https://github.com/appscode/voyager/tree/4.0.0-rc.0) (2017-09-24)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.1...4.0.0-rc.0)

**Implemented enhancements:**

- Don't require spec.providerCredentialSecretName for own provider [\#366](https://github.com/appscode/voyager/issues/366)
- Reimplement certificate controller [\#506](https://github.com/appscode/voyager/pull/506) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- AWS secrets don't seem to be used for ACME validation [\#526](https://github.com/appscode/voyager/issues/526)
- Avoid concurrency for NewACMEClient [\#382](https://github.com/appscode/voyager/issues/382)
- ProviderCredential has to be created before Certificate object [\#370](https://github.com/appscode/voyager/issues/370)
- Watcher should exit if it can't connect to master [\#136](https://github.com/appscode/voyager/issues/136)

**Closed issues:**

- Redesign Certificate CRD [\#505](https://github.com/appscode/voyager/issues/505)
- Upgrade haproxy\_exporter to 0.8.0 [\#504](https://github.com/appscode/voyager/issues/504)
- Support providing secrets as a PV [\#496](https://github.com/appscode/voyager/issues/496)
- Use SharedInformer [\#443](https://github.com/appscode/voyager/issues/443)
- GCE: Services \(LoadBalancer\) with static ip causes panic in 1.7 [\#416](https://github.com/appscode/voyager/issues/416)
- se fields service.spec.externalTrafficPolicy and service.spec.healthCheckNodePort instead [\#415](https://github.com/appscode/voyager/issues/415)
- Validate certificates [\#393](https://github.com/appscode/voyager/issues/393)
- Don't retry if rate-limited by LE [\#356](https://github.com/appscode/voyager/issues/356)
- Use OwnerReference [\#285](https://github.com/appscode/voyager/issues/285)

**Merged pull requests:**

- Add owner reference for Ingress [\#530](https://github.com/appscode/voyager/pull/530) ([tamalsaha](https://github.com/tamalsaha))
- Fix install guide link. [\#523](https://github.com/appscode/voyager/pull/523) ([tamalsaha](https://github.com/tamalsaha))
- Add HAProxy 1.7.9 [\#522](https://github.com/appscode/voyager/pull/522) ([tamalsaha](https://github.com/tamalsaha))
- Add e2e test for HSTS annotations [\#521](https://github.com/appscode/voyager/pull/521) ([diptadas](https://github.com/diptadas))
- Fix HSTS header template [\#520](https://github.com/appscode/voyager/pull/520) ([diptadas](https://github.com/diptadas))
- Add hsts-preload and hsts-include-subdomains annotations [\#519](https://github.com/appscode/voyager/pull/519) ([diptadas](https://github.com/diptadas))
- Update kloader to 4.0.1 [\#518](https://github.com/appscode/voyager/pull/518) ([tamalsaha](https://github.com/tamalsaha))
- Add hsts-max-age annotation [\#515](https://github.com/appscode/voyager/pull/515) ([diptadas](https://github.com/diptadas))
- Revendor haproxy-exporter [\#513](https://github.com/appscode/voyager/pull/513) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.1](https://github.com/appscode/voyager/tree/3.2.1) (2017-09-19)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-alpha.1...3.2.1)

**Implemented enhancements:**

- Fix HTTP Provider Certificate [\#502](https://github.com/appscode/voyager/pull/502) ([sadlil](https://github.com/sadlil))
- Add ssl passthrough support for annotations [\#501](https://github.com/appscode/voyager/pull/501) ([sadlil](https://github.com/sadlil))
- Add Max Body size and CORS annotations [\#500](https://github.com/appscode/voyager/pull/500) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Update RBAC to allow watching nodes. [\#510](https://github.com/appscode/voyager/pull/510) ([tamalsaha](https://github.com/tamalsaha))
- Fix DNS provider key for Google cloud DNS. [\#509](https://github.com/appscode/voyager/pull/509) ([tamalsaha](https://github.com/tamalsaha))
- Change HAProxy image tag to 1.7.6-4.0.0-alpha.1 [\#499](https://github.com/appscode/voyager/pull/499) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-alpha.1](https://github.com/appscode/voyager/tree/4.0.0-alpha.1) (2017-09-15)
[Full Changelog](https://github.com/appscode/voyager/compare/4.0.0-alpha.0...4.0.0-alpha.1)

**Implemented enhancements:**

- Add support for affinity annotations for ingress [\#493](https://github.com/appscode/voyager/pull/493) ([sadlil](https://github.com/sadlil))
- Fix tests for 4.0 [\#492](https://github.com/appscode/voyager/pull/492) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Allow configuring templates per Ingress [\#482](https://github.com/appscode/voyager/issues/482)
- Bring annotation parity with Nginx Ingress [\#278](https://github.com/appscode/voyager/issues/278)
- Update GCP annotation for preserving source IP [\#276](https://github.com/appscode/voyager/issues/276)

**Merged pull requests:**

- Use kloader 4.0.0 [\#498](https://github.com/appscode/voyager/pull/498) ([tamalsaha](https://github.com/tamalsaha))
- Add support for `ingress.kubernetes.io/session-cookie-hash`. [\#497](https://github.com/appscode/voyager/pull/497) ([sadlil](https://github.com/sadlil))
- Correct a small typo in the weighted doco [\#495](https://github.com/appscode/voyager/pull/495) ([leprechaun](https://github.com/leprechaun))
- Add ObjectReference methods. [\#494](https://github.com/appscode/voyager/pull/494) ([tamalsaha](https://github.com/tamalsaha))
- Update Chart RBAC format as recommended. [\#490](https://github.com/appscode/voyager/pull/490) ([tamalsaha](https://github.com/tamalsaha))

## [4.0.0-alpha.0](https://github.com/appscode/voyager/tree/4.0.0-alpha.0) (2017-09-11)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0...4.0.0-alpha.0)

**Implemented enhancements:**

- Replace TPR with CRD [\#419](https://github.com/appscode/voyager/pull/419) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Use Deployment for HostPort mode [\#446](https://github.com/appscode/voyager/issues/446)
- Set DNSpolicy to ClusterFirstWithHostNet in HostPort mode [\#417](https://github.com/appscode/voyager/issues/417)
- Use kubernetes/code-generator to generate clients [\#329](https://github.com/appscode/voyager/issues/329)
- Switch to CustomResourceDefinitions [\#239](https://github.com/appscode/voyager/issues/239)
- Use Deployments from apps/v1beta1 [\#238](https://github.com/appscode/voyager/issues/238)

**Merged pull requests:**

- Use svc.Spec.ExternalTrafficPolicy [\#489](https://github.com/appscode/voyager/pull/489) ([tamalsaha](https://github.com/tamalsaha))
- Use DNSPolicy ClusterFirstWithHostNet for HostPort mode. [\#488](https://github.com/appscode/voyager/pull/488) ([tamalsaha](https://github.com/tamalsaha))
- Use log & errors to appscode/go pkg [\#487](https://github.com/appscode/voyager/pull/487) ([tamalsaha](https://github.com/tamalsaha))
- Use Deployment for HostPort mode [\#486](https://github.com/appscode/voyager/pull/486) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0](https://github.com/appscode/voyager/tree/3.2.0) (2017-09-11)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-rc.3...3.2.0)

**Implemented enhancements:**

- haproxy stats, named services [\#310](https://github.com/appscode/voyager/issues/310)
- Serve both HTTP and HTTPS under same host [\#262](https://github.com/appscode/voyager/issues/262)
- Open firewall for know providers in NodePort mode. [\#227](https://github.com/appscode/voyager/issues/227)
- Allow users to specify NodePort for service ports in NodePort mode. [\#128](https://github.com/appscode/voyager/issues/128)
- Run L7 ingress on non-standard ports [\#73](https://github.com/appscode/voyager/issues/73)
- Validate Ingress [\#46](https://github.com/appscode/voyager/issues/46)
- Update 3.2.0 Docs [\#477](https://github.com/appscode/voyager/pull/477) ([sadlil](https://github.com/sadlil))
- Implement Basic Auth for HTTP Ingresses [\#470](https://github.com/appscode/voyager/pull/470) ([sadlil](https://github.com/sadlil))
- Frontend rules [\#467](https://github.com/appscode/voyager/pull/467) ([sadlil](https://github.com/sadlil))
- Apply Session affinity to Backend service [\#460](https://github.com/appscode/voyager/pull/460) ([sadlil](https://github.com/sadlil))
- Restart HAProxy in case of renew certificates [\#413](https://github.com/appscode/voyager/pull/413) ([sadlil](https://github.com/sadlil))
- Converting E2E tests to use Ginkgo [\#334](https://github.com/appscode/voyager/pull/334) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Ingress validation error [\#420](https://github.com/appscode/voyager/issues/420)
- Fix ACL for host:port in non-standard ports. [\#418](https://github.com/appscode/voyager/issues/418)
- Update operations delete HAProxy pods gets reverted [\#386](https://github.com/appscode/voyager/issues/386)
- Deleting and re-createing a Voyager Ingress in AWS fails due to leaked security groups [\#372](https://github.com/appscode/voyager/issues/372)
- LE cert failed to issue with route53 [\#371](https://github.com/appscode/voyager/issues/371)
- Restart HAProxy when new cert is issued. [\#340](https://github.com/appscode/voyager/issues/340)
- Cert controller issues [\#124](https://github.com/appscode/voyager/issues/124)
- Automatically update firewall when nodeSelector is changed. [\#20](https://github.com/appscode/voyager/issues/20)
- Fix SG group name for GCE [\#472](https://github.com/appscode/voyager/pull/472) ([tamalsaha](https://github.com/tamalsaha))
- Correctly detect APISchema\(\) [\#471](https://github.com/appscode/voyager/pull/471) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Bug: stats.cfg generates an extra \t when no auth given [\#480](https://github.com/appscode/voyager/issues/480)
- 3.2.0 docs [\#474](https://github.com/appscode/voyager/issues/474)
- Allow Sticky session per service basis [\#453](https://github.com/appscode/voyager/issues/453)
- Document how to whitelist IPs [\#441](https://github.com/appscode/voyager/issues/441)
- Allow configuring logging [\#439](https://github.com/appscode/voyager/issues/439)
- Add PATCH api support [\#411](https://github.com/appscode/voyager/issues/411)
- Handle SSL frontend and backends [\#396](https://github.com/appscode/voyager/issues/396)
- Set unit for timeouts in template [\#360](https://github.com/appscode/voyager/issues/360)
- Add tests [\#357](https://github.com/appscode/voyager/issues/357)
- Handle errors for serviceEndpoints\(\) and getEndpoints\(\) [\#350](https://github.com/appscode/voyager/issues/350)
- Split ingress controller into micro controllers [\#347](https://github.com/appscode/voyager/issues/347)
- setting  a static port for type nodeport  [\#344](https://github.com/appscode/voyager/issues/344)
- Allow option http-keep-alive and TLS backends [\#343](https://github.com/appscode/voyager/issues/343)
- Open port 443 in HTTP mode [\#333](https://github.com/appscode/voyager/issues/333)
- Revise TCP secret name [\#319](https://github.com/appscode/voyager/issues/319)
- Show validation error if multiple TCP rules are sharing the same port [\#318](https://github.com/appscode/voyager/issues/318)
- Clean up cert controller. [\#287](https://github.com/appscode/voyager/issues/287)
- Improve Prometheus labels from HAProxy Exporter [\#271](https://github.com/appscode/voyager/issues/271)
- Convert tests to use Ginkgo [\#257](https://github.com/appscode/voyager/issues/257)
- Add tests for TLS [\#175](https://github.com/appscode/voyager/issues/175)
- Correctly compute content hash for HAproxy config [\#138](https://github.com/appscode/voyager/issues/138)
- Improve test suite [\#31](https://github.com/appscode/voyager/issues/31)

**Merged pull requests:**

- Document noTLS feature [\#485](https://github.com/appscode/voyager/pull/485) ([tamalsaha](https://github.com/tamalsaha))
- Keep whitespae from end to templates in haproxy.cfg [\#483](https://github.com/appscode/voyager/pull/483) ([tamalsaha](https://github.com/tamalsaha))
- Fix stats auth indentation when auth is omitted [\#481](https://github.com/appscode/voyager/pull/481) ([julianvmodesto](https://github.com/julianvmodesto))
- Fix typo in doc [\#479](https://github.com/appscode/voyager/pull/479) ([pierreozoux](https://github.com/pierreozoux))
- Fix links in docs [\#478](https://github.com/appscode/voyager/pull/478) ([pierreozoux](https://github.com/pierreozoux))
- Prepare docs for 3.2.0 [\#476](https://github.com/appscode/voyager/pull/476) ([tamalsaha](https://github.com/tamalsaha))
- Enable accept-proxy [\#475](https://github.com/appscode/voyager/pull/475) ([tamalsaha](https://github.com/tamalsaha))
- Document how to use custom templates for HAProxy [\#462](https://github.com/appscode/voyager/pull/462) ([tamalsaha](https://github.com/tamalsaha))
- Fix NPE [\#469](https://github.com/appscode/voyager/pull/469) ([tamalsaha](https://github.com/tamalsaha))
- Use .cfg extension for templates. [\#465](https://github.com/appscode/voyager/pull/465) ([tamalsaha](https://github.com/tamalsaha))
- Modify certificate docs. [\#463](https://github.com/appscode/voyager/pull/463) ([sadlil](https://github.com/sadlil))
- Support custom user templates [\#454](https://github.com/appscode/voyager/pull/454) ([tamalsaha](https://github.com/tamalsaha))
- Add ingress.appscode.com/accept-proxy annotation [\#452](https://github.com/appscode/voyager/pull/452) ([tamalsaha](https://github.com/tamalsaha))
- Update client-go to 3.0.0 from 3.0.0-beta [\#406](https://github.com/appscode/voyager/pull/406) ([tamalsaha](https://github.com/tamalsaha))
- Update Azure SDK to 10.2.1-beta [\#402](https://github.com/appscode/voyager/pull/402) ([tamalsaha](https://github.com/tamalsaha))
- Assign VoyagerCluster tag for Voyager Ingress [\#401](https://github.com/appscode/voyager/pull/401) ([tamalsaha](https://github.com/tamalsaha))
- Check for unset env var passed as flag values. [\#399](https://github.com/appscode/voyager/pull/399) ([tamalsaha](https://github.com/tamalsaha))
- Merge service and pod annotations [\#390](https://github.com/appscode/voyager/pull/390) ([tamalsaha](https://github.com/tamalsaha))
- Maintain support for Kubernetes 1.5 for HostPort daemonsets [\#388](https://github.com/appscode/voyager/pull/388) ([tamalsaha](https://github.com/tamalsaha))
- Split ingress controller into micro controllers [\#383](https://github.com/appscode/voyager/pull/383) ([tamalsaha](https://github.com/tamalsaha))
- Fix GO reportcard issues. [\#379](https://github.com/appscode/voyager/pull/379) ([tamalsaha](https://github.com/tamalsaha))
- Add voyager check command. [\#364](https://github.com/appscode/voyager/pull/364) ([tamalsaha](https://github.com/tamalsaha))
- Update Ingress spec [\#317](https://github.com/appscode/voyager/pull/317) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0-rc.3](https://github.com/appscode/voyager/tree/3.2.0-rc.3) (2017-09-07)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-rc.2...3.2.0-rc.3)

**Closed issues:**

- Fix NodePort docs [\#461](https://github.com/appscode/voyager/issues/461)

**Merged pull requests:**

- Update NodePort docs [\#466](https://github.com/appscode/voyager/pull/466) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0-rc.2](https://github.com/appscode/voyager/tree/3.2.0-rc.2) (2017-09-06)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-rc.1...3.2.0-rc.2)

**Fixed bugs:**

- OVH DNS provider is not working [\#449](https://github.com/appscode/voyager/issues/449)
- bug: ServiceAccount does not exist after upgrading [\#448](https://github.com/appscode/voyager/issues/448)

**Closed issues:**

- `keep-source-ip` should enable PROXY protocol is bare metal cluster [\#451](https://github.com/appscode/voyager/issues/451)
- Allow users to provide custom templates [\#444](https://github.com/appscode/voyager/issues/444)

**Merged pull requests:**

- Create RBAC objects if missing [\#458](https://github.com/appscode/voyager/pull/458) ([tamalsaha](https://github.com/tamalsaha))
- Move analytics collector to root command [\#450](https://github.com/appscode/voyager/pull/450) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0-rc.1](https://github.com/appscode/voyager/tree/3.2.0-rc.1) (2017-09-01)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-rc.0...3.2.0-rc.1)

**Fixed bugs:**

- Don't sort HTTP paths since the order matters to HAProxy [\#445](https://github.com/appscode/voyager/pull/445) ([tamalsaha](https://github.com/tamalsaha))

**Closed issues:**

- Handle both TCP and HTTP requests on same frontend [\#430](https://github.com/appscode/voyager/issues/430)

**Merged pull requests:**

- Show how to use kubectl. [\#442](https://github.com/appscode/voyager/pull/442) ([tamalsaha](https://github.com/tamalsaha))
- Add Docs [\#438](https://github.com/appscode/voyager/pull/438) ([sadlil](https://github.com/sadlil))
- Fix secret name [\#434](https://github.com/appscode/voyager/pull/434) ([rstuven](https://github.com/rstuven))
- Fix secret name [\#433](https://github.com/appscode/voyager/pull/433) ([rstuven](https://github.com/rstuven))
- Minor fix [\#432](https://github.com/appscode/voyager/pull/432) ([rstuven](https://github.com/rstuven))
- Fix load-balancer-ip annotation references [\#431](https://github.com/appscode/voyager/pull/431) ([rstuven](https://github.com/rstuven))

## [3.2.0-rc.0](https://github.com/appscode/voyager/tree/3.2.0-rc.0) (2017-08-28)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-beta.4...3.2.0-rc.0)

**Fixed bugs:**

- Fix Host:Port Matching issue. [\#425](https://github.com/appscode/voyager/pull/425) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Restart HAProxy in case of renew certificates [\#427](https://github.com/appscode/voyager/pull/427) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 3.2.0-rc.0 [\#426](https://github.com/appscode/voyager/pull/426) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0-beta.4](https://github.com/appscode/voyager/tree/3.2.0-beta.4) (2017-08-27)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-beta.3...3.2.0-beta.4)

**Implemented enhancements:**

- Add Patch API Supports [\#412](https://github.com/appscode/voyager/pull/412) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Fix Ingress validation error [\#421](https://github.com/appscode/voyager/pull/421) ([tamalsaha](https://github.com/tamalsaha))
- Fix cert [\#410](https://github.com/appscode/voyager/pull/410) ([sadlil](https://github.com/sadlil))
- Print back ingress in YAML format [\#409](https://github.com/appscode/voyager/pull/409) ([tamalsaha](https://github.com/tamalsaha))
- TLS Backend [\#408](https://github.com/appscode/voyager/pull/408) ([sadlil](https://github.com/sadlil))

## [3.2.0-beta.3](https://github.com/appscode/voyager/tree/3.2.0-beta.3) (2017-08-19)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-beta.2...3.2.0-beta.3)

**Implemented enhancements:**

- Allow custom options [\#403](https://github.com/appscode/voyager/pull/403) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- single static port for the ingress resource and not a particular service [\#404](https://github.com/appscode/voyager/issues/404)

**Merged pull requests:**

- Improve test suite  [\#394](https://github.com/appscode/voyager/pull/394) ([sadlil](https://github.com/sadlil))

## [3.2.0-beta.2](https://github.com/appscode/voyager/tree/3.2.0-beta.2) (2017-08-16)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-beta.1...3.2.0-beta.2)

## [3.2.0-beta.1](https://github.com/appscode/voyager/tree/3.2.0-beta.1) (2017-08-16)
[Full Changelog](https://github.com/appscode/voyager/compare/3.2.0-beta.0...3.2.0-beta.1)

**Merged pull requests:**

- Change ingress sg tag to VoyagerCluster from KubernetesCluster [\#397](https://github.com/appscode/voyager/pull/397) ([tamalsaha](https://github.com/tamalsaha))
- Remove links to forum [\#395](https://github.com/appscode/voyager/pull/395) ([tamalsaha](https://github.com/tamalsaha))
- Open firewall for know providers in NodePort mode [\#392](https://github.com/appscode/voyager/pull/392) ([tamalsaha](https://github.com/tamalsaha))

## [3.2.0-beta.0](https://github.com/appscode/voyager/tree/3.2.0-beta.0) (2017-08-14)
[Full Changelog](https://github.com/appscode/voyager/compare/3.1.4...3.2.0-beta.0)

**Closed issues:**

- Validate existing Ingress before starting operator [\#346](https://github.com/appscode/voyager/issues/346)

**Merged pull requests:**

- Make AWS HostPort SG name unique across clusters [\#391](https://github.com/appscode/voyager/pull/391) ([tamalsaha](https://github.com/tamalsaha))
- Fix AWS SecurityGroup leakage in HostPort mode [\#389](https://github.com/appscode/voyager/pull/389) ([tamalsaha](https://github.com/tamalsaha))
- Revise ingress controller update operations	 [\#385](https://github.com/appscode/voyager/pull/385) ([tamalsaha](https://github.com/tamalsaha))
- Split IsExists tests [\#384](https://github.com/appscode/voyager/pull/384) ([tamalsaha](https://github.com/tamalsaha))
- Update aws sdk to v1.6.10 [\#381](https://github.com/appscode/voyager/pull/381) ([tamalsaha](https://github.com/tamalsaha))
- Avoid getting provider secret [\#378](https://github.com/appscode/voyager/pull/378) ([sadlil](https://github.com/sadlil))
- Fix BUGS and Tests [\#363](https://github.com/appscode/voyager/pull/363) ([sadlil](https://github.com/sadlil))

## [3.1.4](https://github.com/appscode/voyager/tree/3.1.4) (2017-08-11)
[Full Changelog](https://github.com/appscode/voyager/compare/3.1.3...3.1.4)

**Closed issues:**

- Test aws cert manager 80-\>443 redirect [\#353](https://github.com/appscode/voyager/issues/353)

**Merged pull requests:**

- Revendor lego [\#377](https://github.com/appscode/voyager/pull/377) ([tamalsaha](https://github.com/tamalsaha))
- Detect port changes correctly. [\#376](https://github.com/appscode/voyager/pull/376) ([tamalsaha](https://github.com/tamalsaha))
- Revendor lego to detect DNS zone correctly. [\#375](https://github.com/appscode/voyager/pull/375) ([tamalsaha](https://github.com/tamalsaha))
- Revendor lego [\#373](https://github.com/appscode/voyager/pull/373) ([tamalsaha](https://github.com/tamalsaha))
- Fix Implicit timeouts [\#361](https://github.com/appscode/voyager/pull/361) ([sadlil](https://github.com/sadlil))

## [3.1.3](https://github.com/appscode/voyager/tree/3.1.3) (2017-08-08)
[Full Changelog](https://github.com/appscode/voyager/compare/3.1.2...3.1.3)

**Fixed bugs:**

- Fix Event Recorder type [\#341](https://github.com/appscode/voyager/pull/341) ([sadlil](https://github.com/sadlil))
- Fix Domain Comparison  [\#339](https://github.com/appscode/voyager/pull/339) ([sadlil](https://github.com/sadlil))
- Allow secret create/update for Voyager cert controller. [\#338](https://github.com/appscode/voyager/pull/338) ([tamalsaha](https://github.com/tamalsaha))

**Merged pull requests:**

- Fix test docs for ginkgo tests [\#352](https://github.com/appscode/voyager/pull/352) ([sadlil](https://github.com/sadlil))
- Add DCO [\#351](https://github.com/appscode/voyager/pull/351) ([tamalsaha](https://github.com/tamalsaha))
- Rename Ingress controller receiver to c from lbc [\#345](https://github.com/appscode/voyager/pull/345) ([tamalsaha](https://github.com/tamalsaha))

## [3.1.2](https://github.com/appscode/voyager/tree/3.1.2) (2017-08-02)
[Full Changelog](https://github.com/appscode/voyager/compare/3.1.1...3.1.2)

**Implemented enhancements:**

- Use Lets Encrypt Prod URL as default [\#335](https://github.com/appscode/voyager/pull/335) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Use Lets Encrypt Prod URL as default [\#335](https://github.com/appscode/voyager/pull/335) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Prepare docs for 3.1.2 release. [\#336](https://github.com/appscode/voyager/pull/336) ([tamalsaha](https://github.com/tamalsaha))
- Add install scripts [\#332](https://github.com/appscode/voyager/pull/332) ([tamalsaha](https://github.com/tamalsaha))

## [3.1.1](https://github.com/appscode/voyager/tree/3.1.1) (2017-07-22)
[Full Changelog](https://github.com/appscode/voyager/compare/3.1.0...3.1.1)

**Merged pull requests:**

- typos [\#325](https://github.com/appscode/voyager/pull/325) ([nstott](https://github.com/nstott))
- Prepare docs for 3.1.1 release. [\#328](https://github.com/appscode/voyager/pull/328) ([tamalsaha](https://github.com/tamalsaha))
- Add cloud provider specific install scripts. [\#327](https://github.com/appscode/voyager/pull/327) ([tamalsaha](https://github.com/tamalsaha))
- Disable critical addon feature [\#326](https://github.com/appscode/voyager/pull/326) ([tamalsaha](https://github.com/tamalsaha))

## [3.1.0](https://github.com/appscode/voyager/tree/3.1.0) (2017-07-21)
[Full Changelog](https://github.com/appscode/voyager/compare/3.0.0...3.1.0)

**Implemented enhancements:**

- Record events against TPR [\#79](https://github.com/appscode/voyager/issues/79)
- Remove event framework from certificate [\#284](https://github.com/appscode/voyager/pull/284) ([sadlil](https://github.com/sadlil))
- Fix RBAC configs [\#295](https://github.com/appscode/voyager/pull/295) ([sadlil](https://github.com/sadlil))
- Add configure option for Haproxy default timeout. [\#286](https://github.com/appscode/voyager/pull/286) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- podAffinityTerm.topologyKey: Required value: can not be empty [\#320](https://github.com/appscode/voyager/issues/320)
- Restore objects if deleted by mistake. [\#283](https://github.com/appscode/voyager/issues/283)
- HostPort mode does not work for AWS [\#281](https://github.com/appscode/voyager/issues/281)
- Externalservice redirection gets reset [\#279](https://github.com/appscode/voyager/issues/279)
- Voyager doesn't work with cloud = minikube and type = HostPort [\#272](https://github.com/appscode/voyager/issues/272)
- Adding cert manager to existing ingress does not open port 443 [\#267](https://github.com/appscode/voyager/issues/267)
- Bug: annotations are not applied [\#266](https://github.com/appscode/voyager/issues/266)
- Add newline in pem file [\#261](https://github.com/appscode/voyager/issues/261)
- Adding SSL to an existing ingress does not mount certs [\#260](https://github.com/appscode/voyager/issues/260)
- Set topology key for pod anti-affinity [\#321](https://github.com/appscode/voyager/pull/321) ([tamalsaha](https://github.com/tamalsaha))
- Correctly detect changed ports [\#322](https://github.com/appscode/voyager/pull/322) ([tamalsaha](https://github.com/tamalsaha))
- Fix Adding SSL to an existing ingress does not mount certs \#260 [\#306](https://github.com/appscode/voyager/pull/306) ([sadlil](https://github.com/sadlil))
- Fix External Service redirect Issue [\#304](https://github.com/appscode/voyager/pull/304) ([sadlil](https://github.com/sadlil))
- Fix RBAC configs [\#295](https://github.com/appscode/voyager/pull/295) ([sadlil](https://github.com/sadlil))
- Fix Operator panic on service restore [\#273](https://github.com/appscode/voyager/pull/273) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Difficulties to setup, scarce docs [\#303](https://github.com/appscode/voyager/issues/303)
- Setup Issues [\#298](https://github.com/appscode/voyager/issues/298)
- Setup Issues [\#297](https://github.com/appscode/voyager/issues/297)
- configurable HAProxy defaults [\#280](https://github.com/appscode/voyager/issues/280)
- Support setting resource for pods [\#277](https://github.com/appscode/voyager/issues/277)
- The link to contribution guide in README.md is broken. [\#274](https://github.com/appscode/voyager/issues/274)
- Voyager exporter sidecar isn't exporting any metrics [\#270](https://github.com/appscode/voyager/issues/270)
- Adding an AWS Cert and opening 80 and 443 doesn't work for plain http:// [\#268](https://github.com/appscode/voyager/issues/268)
- Support HorizontalPodAutoscaling for HAProxy pods [\#242](https://github.com/appscode/voyager/issues/242)
- Test updated chart with RBAC [\#302](https://github.com/appscode/voyager/issues/302)
- Delete TPR when NS is deleted [\#258](https://github.com/appscode/voyager/issues/258)
- voyager-operator should ensure that ServiceAccount/Role/RoleBinding exists for created voyager deploys. [\#252](https://github.com/appscode/voyager/issues/252)
- RBAC objects for Voyager operator. [\#241](https://github.com/appscode/voyager/issues/241)
- Should all hosts be passed to EnsureLoadBalancer [\#88](https://github.com/appscode/voyager/issues/88)

**Merged pull requests:**

- Fix various chart issues [\#324](https://github.com/appscode/voyager/pull/324) ([tamalsaha](https://github.com/tamalsaha))
- Add Custom timeout docs [\#323](https://github.com/appscode/voyager/pull/323) ([sadlil](https://github.com/sadlil))
- Revendor dependencies. [\#312](https://github.com/appscode/voyager/pull/312) ([tamalsaha](https://github.com/tamalsaha))
- move RecognizeWellKnownRegions\(\) to the beginning of newAWSCloud\(\) [\#311](https://github.com/appscode/voyager/pull/311) ([jipperinbham](https://github.com/jipperinbham))
- Add ingress label to exported metrics [\#300](https://github.com/appscode/voyager/pull/300) ([tamalsaha](https://github.com/tamalsaha))
- Support setting resource for pods [\#289](https://github.com/appscode/voyager/pull/289) ([tamalsaha](https://github.com/tamalsaha))
- fix the contribution guild link \(\#274\) [\#275](https://github.com/appscode/voyager/pull/275) ([aimof](https://github.com/aimof))
- Update aws-cert-manager.md [\#269](https://github.com/appscode/voyager/pull/269) ([julianvmodesto](https://github.com/julianvmodesto))
- Add command reference docs [\#265](https://github.com/appscode/voyager/pull/265) ([tamalsaha](https://github.com/tamalsaha))
- Point to HPA example on readme pages. [\#254](https://github.com/appscode/voyager/pull/254) ([tamalsaha](https://github.com/tamalsaha))
- Add example with hpa [\#253](https://github.com/appscode/voyager/pull/253) ([julianvmodesto](https://github.com/julianvmodesto))
- Use ```console instead of ```sh syntax highlighting [\#309](https://github.com/appscode/voyager/pull/309) ([tamalsaha](https://github.com/tamalsaha))
- Install Voyager as critical addon [\#301](https://github.com/appscode/voyager/pull/301) ([tamalsaha](https://github.com/tamalsaha))
- Add Stats Service events [\#299](https://github.com/appscode/voyager/pull/299) ([sadlil](https://github.com/sadlil))
- Recover ServiceMonitor [\#294](https://github.com/appscode/voyager/pull/294) ([tamalsaha](https://github.com/tamalsaha))
- Make node selectors optional for HostPort [\#293](https://github.com/appscode/voyager/pull/293) ([tamalsaha](https://github.com/tamalsaha))
- Delete kube lister classes. [\#291](https://github.com/appscode/voyager/pull/291) ([tamalsaha](https://github.com/tamalsaha))
- Record events against TPR [\#290](https://github.com/appscode/voyager/pull/290) ([tamalsaha](https://github.com/tamalsaha))
- Add tpr constants [\#288](https://github.com/appscode/voyager/pull/288) ([tamalsaha](https://github.com/tamalsaha))
- Remove event framework [\#282](https://github.com/appscode/voyager/pull/282) ([tamalsaha](https://github.com/tamalsaha))
- Update dev docs. [\#264](https://github.com/appscode/voyager/pull/264) ([tamalsaha](https://github.com/tamalsaha))
- Add a newline between crt & key. [\#263](https://github.com/appscode/voyager/pull/263) ([tamalsaha](https://github.com/tamalsaha))
- Create RBAC roles for Voyager during installation [\#256](https://github.com/appscode/voyager/pull/256) ([tamalsaha](https://github.com/tamalsaha))
- Support non-default service account with offshoot pods [\#255](https://github.com/appscode/voyager/pull/255) ([tamalsaha](https://github.com/tamalsaha))

## [3.0.0](https://github.com/appscode/voyager/tree/3.0.0) (2017-06-23)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.6...3.0.0)

**Implemented enhancements:**

- Automatically create ServiceMonitor for built-in exporter [\#154](https://github.com/appscode/voyager/issues/154)
- Fix testframework for aws and update docs. [\#237](https://github.com/appscode/voyager/pull/237) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Delete pods & services matching old labels before starting operator [\#229](https://github.com/appscode/voyager/issues/229)
- Check for updates properly [\#250](https://github.com/appscode/voyager/pull/250) ([tamalsaha](https://github.com/tamalsaha))
- Don't restore stats service if stats is disabled. [\#249](https://github.com/appscode/voyager/pull/249) ([tamalsaha](https://github.com/tamalsaha))
- Apply labels to stats service for service monitor [\#248](https://github.com/appscode/voyager/pull/248) ([tamalsaha](https://github.com/tamalsaha))
- Fix Bugs [\#247](https://github.com/appscode/voyager/pull/247) ([sadlil](https://github.com/sadlil))
- Correctly parse target port [\#245](https://github.com/appscode/voyager/pull/245) ([tamalsaha](https://github.com/tamalsaha))
- Fix testframework for aws and update docs. [\#237](https://github.com/appscode/voyager/pull/237) ([sadlil](https://github.com/sadlil))
- Add dns-resovler-check-health annotation to for ExternalName service [\#226](https://github.com/appscode/voyager/pull/226) ([tamalsaha](https://github.com/tamalsaha))
- Add cloud config file  [\#218](https://github.com/appscode/voyager/pull/218) ([sadlil](https://github.com/sadlil))
- Fix bugs [\#217](https://github.com/appscode/voyager/pull/217) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Add chart value for --cloud-config mount [\#228](https://github.com/appscode/voyager/issues/228)
- Document http-\>https redirect with AWS cert manager [\#225](https://github.com/appscode/voyager/issues/225)
- Update version policy [\#194](https://github.com/appscode/voyager/issues/194)
- Change api group to voyager.appscode.com [\#193](https://github.com/appscode/voyager/issues/193)
- Use client-go [\#192](https://github.com/appscode/voyager/issues/192)
- Use pod anti-affinity for deployments [\#161](https://github.com/appscode/voyager/issues/161)
- Change api group to voyager.appscode.com [\#142](https://github.com/appscode/voyager/issues/142)

**Merged pull requests:**

- Small typo fix \(CLOUDE\_CONFIG =\> CLOUD\_CONFIG\) [\#251](https://github.com/appscode/voyager/pull/251) ([thecodeassassin](https://github.com/thecodeassassin))
- Document http-\>https redirect with AWS cert manager [\#235](https://github.com/appscode/voyager/pull/235) ([tamalsaha](https://github.com/tamalsaha))
- Remove deprecated Daemon type. [\#205](https://github.com/appscode/voyager/pull/205) ([tamalsaha](https://github.com/tamalsaha))
- Automatically create ServiceMonitor for built-in exporter [\#203](https://github.com/appscode/voyager/pull/203) ([tamalsaha](https://github.com/tamalsaha))
- Track operator version [\#200](https://github.com/appscode/voyager/pull/200) ([tamalsaha](https://github.com/tamalsaha))
- Update version policy to point to client-go [\#198](https://github.com/appscode/voyager/pull/198) ([tamalsaha](https://github.com/tamalsaha))
- Use client-go [\#196](https://github.com/appscode/voyager/pull/196) ([tamalsaha](https://github.com/tamalsaha))
- Use stats service port name in ServiceMonitor [\#246](https://github.com/appscode/voyager/pull/246) ([tamalsaha](https://github.com/tamalsaha))
- Use correct api schema when checking ingress class. [\#244](https://github.com/appscode/voyager/pull/244) ([tamalsaha](https://github.com/tamalsaha))
- Note test-ns policy [\#243](https://github.com/appscode/voyager/pull/243) ([tamalsaha](https://github.com/tamalsaha))
- Add acs  provider [\#236](https://github.com/appscode/voyager/pull/236) ([tamalsaha](https://github.com/tamalsaha))
- Update chart readme for cloud config [\#234](https://github.com/appscode/voyager/pull/234) ([tamalsaha](https://github.com/tamalsaha))
- Make cloud config configurable. [\#233](https://github.com/appscode/voyager/pull/233) ([tamalsaha](https://github.com/tamalsaha))
- Change api group to networking.appscode.com [\#232](https://github.com/appscode/voyager/pull/232) ([tamalsaha](https://github.com/tamalsaha))
- Update \*\*\*Getter interfaces match form [\#231](https://github.com/appscode/voyager/pull/231) ([tamalsaha](https://github.com/tamalsaha))
- Delete pods & services matching old labels before starting operator [\#230](https://github.com/appscode/voyager/pull/230) ([tamalsaha](https://github.com/tamalsaha))
- Use PreRun & PostRun to send analytics. [\#224](https://github.com/appscode/voyager/pull/224) ([tamalsaha](https://github.com/tamalsaha))
- Update metric endpoints documentation. [\#223](https://github.com/appscode/voyager/pull/223) ([tamalsaha](https://github.com/tamalsaha))
- Fix port used for exposing metrics from operator. [\#222](https://github.com/appscode/voyager/pull/222) ([tamalsaha](https://github.com/tamalsaha))
- Open both port 443 & 80 when AWS cert manager is in use. [\#221](https://github.com/appscode/voyager/pull/221) ([tamalsaha](https://github.com/tamalsaha))
- Mount cloud config in chart [\#220](https://github.com/appscode/voyager/pull/220) ([tamalsaha](https://github.com/tamalsaha))
- Use root user inside docker [\#219](https://github.com/appscode/voyager/pull/219) ([tamalsaha](https://github.com/tamalsaha))
- Rename exporter port to targetPort [\#216](https://github.com/appscode/voyager/pull/216) ([tamalsaha](https://github.com/tamalsaha))
- Use Voyager group name correctly. [\#215](https://github.com/appscode/voyager/pull/215) ([tamalsaha](https://github.com/tamalsaha))
- Update default ports [\#214](https://github.com/appscode/voyager/pull/214) ([tamalsaha](https://github.com/tamalsaha))
- Update docs for service monitor integration [\#213](https://github.com/appscode/voyager/pull/213) ([tamalsaha](https://github.com/tamalsaha))
- Fix unit test build issues [\#210](https://github.com/appscode/voyager/pull/210) ([tamalsaha](https://github.com/tamalsaha))
- Change api group to voyager.appscode.com [\#209](https://github.com/appscode/voyager/pull/209) ([tamalsaha](https://github.com/tamalsaha))
- Update docs to point to 3.0.0 [\#208](https://github.com/appscode/voyager/pull/208) ([tamalsaha](https://github.com/tamalsaha))
- Stop creating stats service. [\#207](https://github.com/appscode/voyager/pull/207) ([tamalsaha](https://github.com/tamalsaha))
- Update labels applied to HAProxy pods & services. [\#206](https://github.com/appscode/voyager/pull/206) ([tamalsaha](https://github.com/tamalsaha))
- Fix client-go fake import [\#204](https://github.com/appscode/voyager/pull/204) ([tamalsaha](https://github.com/tamalsaha))
- Change default HAProxy image to 1.7.6-3.0.0 [\#202](https://github.com/appscode/voyager/pull/202) ([tamalsaha](https://github.com/tamalsaha))
- Add HAProxy 1.7.6 dockerfiles. [\#201](https://github.com/appscode/voyager/pull/201) ([tamalsaha](https://github.com/tamalsaha))
- Add voyager export command [\#199](https://github.com/appscode/voyager/pull/199) ([tamalsaha](https://github.com/tamalsaha))
- Only keep Firewall\(\) interface in cloud provider [\#195](https://github.com/appscode/voyager/pull/195) ([tamalsaha](https://github.com/tamalsaha))

## [1.5.6](https://github.com/appscode/voyager/tree/1.5.6) (2017-06-16)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.5...1.5.6)

**Implemented enhancements:**

- Delete docker image from docker hub after integration test [\#125](https://github.com/appscode/voyager/issues/125)
- Change how stats work [\#106](https://github.com/appscode/voyager/issues/106)
- Use AWS ELB Proxy Protocol [\#100](https://github.com/appscode/voyager/issues/100)
- Track Kube's refactoring cloud provider API [\#36](https://github.com/appscode/voyager/issues/36)
- Expose HAProxy stats to prometheus [\#13](https://github.com/appscode/voyager/issues/13)
- Support AWS cert manager [\#189](https://github.com/appscode/voyager/pull/189) ([tamalsaha](https://github.com/tamalsaha))
- Merge existing pods and service during create ingress resource [\#181](https://github.com/appscode/voyager/pull/181) ([sadlil](https://github.com/sadlil))
- Add support for ServiceTypeExternalName [\#167](https://github.com/appscode/voyager/pull/167) ([sadlil](https://github.com/sadlil))
- Collect analytics for voyager usages [\#133](https://github.com/appscode/voyager/pull/133) ([sadlil](https://github.com/sadlil))
- Fix stats behavior [\#130](https://github.com/appscode/voyager/pull/130) ([sadlil](https://github.com/sadlil))
- Improve test framework [\#121](https://github.com/appscode/voyager/pull/121) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Error out if Daemon type does not provide a node selector. [\#159](https://github.com/appscode/voyager/issues/159)
- Disable analytics when running tests [\#147](https://github.com/appscode/voyager/issues/147)
- Missing services should be an warning not error stack error [\#137](https://github.com/appscode/voyager/issues/137)
- Bad ingress object results in unstable HAProxy [\#135](https://github.com/appscode/voyager/issues/135)
- Add ingress hostname to Ingress [\#132](https://github.com/appscode/voyager/issues/132)
- Deleting LB deployment does not get recreated [\#123](https://github.com/appscode/voyager/issues/123)
- Ensure HAproxy running when endpoints changes. [\#120](https://github.com/appscode/voyager/issues/120)
- Updating Ingress annotations are not picked up by controller [\#115](https://github.com/appscode/voyager/issues/115)
- Fix Ingress Status Update Properly. [\#134](https://github.com/appscode/voyager/pull/134) ([sadlil](https://github.com/sadlil))
- Expose monitoring port in chart and deploy yamls [\#156](https://github.com/appscode/voyager/pull/156) ([tamalsaha](https://github.com/tamalsaha))
- Add LoadBalancerSourceRange to ingress Spec [\#148](https://github.com/appscode/voyager/pull/148) ([sadlil](https://github.com/sadlil))
- Ensure loadbalancer resource [\#145](https://github.com/appscode/voyager/pull/145) ([sadlil](https://github.com/sadlil))
- Add annotation to add accept-proxy in bind statements [\#144](https://github.com/appscode/voyager/pull/144) ([sadlil](https://github.com/sadlil))
- Remove unwanted stacktrace from log [\#139](https://github.com/appscode/voyager/pull/139) ([sadlil](https://github.com/sadlil))
- Fix stats behavior [\#130](https://github.com/appscode/voyager/pull/130) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Allow exposing port 443 on the LoadBalancer Service [\#188](https://github.com/appscode/voyager/issues/188)
- Source IP detection  [\#146](https://github.com/appscode/voyager/issues/146)
- helm chart [\#113](https://github.com/appscode/voyager/issues/113)
- Merge pods & services even on create [\#172](https://github.com/appscode/voyager/issues/172)
- Document 1.5.6 changes [\#150](https://github.com/appscode/voyager/issues/150)
- Support Services of type ExternalName [\#127](https://github.com/appscode/voyager/issues/127)
- Collect usage analytics [\#126](https://github.com/appscode/voyager/issues/126)
- Support use of field spec.loadBalancerSourceRanges on Services of type LoadBalancer [\#122](https://github.com/appscode/voyager/issues/122)

**Merged pull requests:**

- Fix chart path [\#191](https://github.com/appscode/voyager/pull/191) ([tamalsaha](https://github.com/tamalsaha))
-  ./hack/make.py test\_deploy to generate deploymnts yaml [\#184](https://github.com/appscode/voyager/pull/184) ([ashiquzzaman33](https://github.com/ashiquzzaman33))
- Disable analytics for test runs [\#182](https://github.com/appscode/voyager/pull/182) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for 1.5.6 [\#178](https://github.com/appscode/voyager/pull/178) ([tamalsaha](https://github.com/tamalsaha))
- Remove cluster name flag [\#177](https://github.com/appscode/voyager/pull/177) ([tamalsaha](https://github.com/tamalsaha))
- Remove persist annotation [\#174](https://github.com/appscode/voyager/pull/174) ([tamalsaha](https://github.com/tamalsaha))
- Add TLS certs for testing [\#173](https://github.com/appscode/voyager/pull/173) ([tamalsaha](https://github.com/tamalsaha))
- Run kloader check without exec [\#171](https://github.com/appscode/voyager/pull/171) ([tamalsaha](https://github.com/tamalsaha))
- Error out if Daemon type does not provide a node selector. [\#168](https://github.com/appscode/voyager/pull/168) ([tamalsaha](https://github.com/tamalsaha))
- Remove dependency on k8s-addons [\#141](https://github.com/appscode/voyager/pull/141) ([tamalsaha](https://github.com/tamalsaha))
- Use kloader 1.5.1 and check config before starting runit. [\#140](https://github.com/appscode/voyager/pull/140) ([tamalsaha](https://github.com/tamalsaha))
- Use ci-space cluster for testing [\#131](https://github.com/appscode/voyager/pull/131) ([ashiquzzaman33](https://github.com/ashiquzzaman33))
- tcp.md: fix typo/port mismatch [\#119](https://github.com/appscode/voyager/pull/119) ([alekssaul](https://github.com/alekssaul))
- Add Jenkinsfile [\#118](https://github.com/appscode/voyager/pull/118) ([ashiquzzaman33](https://github.com/ashiquzzaman33))
- Jenkins test patch1 [\#117](https://github.com/appscode/voyager/pull/117) ([ashiquzzaman33](https://github.com/ashiquzzaman33))
- Document flag options [\#190](https://github.com/appscode/voyager/pull/190) ([tamalsaha](https://github.com/tamalsaha))
- Docs for 1.5.6 [\#183](https://github.com/appscode/voyager/pull/183) ([sadlil](https://github.com/sadlil))
- Set metrics port to :8080 by default [\#180](https://github.com/appscode/voyager/pull/180) ([tamalsaha](https://github.com/tamalsaha))
- Stop redefining -h flag for run command. [\#179](https://github.com/appscode/voyager/pull/179) ([tamalsaha](https://github.com/tamalsaha))
- Remove --cluster-name flag [\#176](https://github.com/appscode/voyager/pull/176) ([tamalsaha](https://github.com/tamalsaha))
- Add nil check before reading options from Ingress annotations. [\#170](https://github.com/appscode/voyager/pull/170) ([tamalsaha](https://github.com/tamalsaha))
- Various cleanup of annotations [\#169](https://github.com/appscode/voyager/pull/169) ([tamalsaha](https://github.com/tamalsaha))
- Use hyphen separated words as annotation key. [\#166](https://github.com/appscode/voyager/pull/166) ([tamalsaha](https://github.com/tamalsaha))
- Use ingress.appscode.com/keep-source-ip: true to preserve source IP [\#165](https://github.com/appscode/voyager/pull/165) ([tamalsaha](https://github.com/tamalsaha))
- Combine annotation keys ip & persist into persist [\#162](https://github.com/appscode/voyager/pull/162) ([tamalsaha](https://github.com/tamalsaha))
- Make nodeSelector annotation applicable for any mode. [\#160](https://github.com/appscode/voyager/pull/160) ([tamalsaha](https://github.com/tamalsaha))
- Explain versioning policy. [\#158](https://github.com/appscode/voyager/pull/158) ([tamalsaha](https://github.com/tamalsaha))
- Apply various comments from official charts team [\#157](https://github.com/appscode/voyager/pull/157) ([tamalsaha](https://github.com/tamalsaha))
- Move component docs directly under user-guide [\#155](https://github.com/appscode/voyager/pull/155) ([tamalsaha](https://github.com/tamalsaha))
- Expose Operator & HAProxy metrics [\#153](https://github.com/appscode/voyager/pull/153) ([tamalsaha](https://github.com/tamalsaha))
- Reorganize code to add run sub command [\#152](https://github.com/appscode/voyager/pull/152) ([tamalsaha](https://github.com/tamalsaha))
- Add forked cloudprovider in third\_party package [\#151](https://github.com/appscode/voyager/pull/151) ([tamalsaha](https://github.com/tamalsaha))

## [1.5.5](https://github.com/appscode/voyager/tree/1.5.5) (2017-05-22)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.4...1.5.5)

**Implemented enhancements:**

- Support user provided annotations [\#103](https://github.com/appscode/voyager/issues/103)
- Rename Daemon type to HostPort [\#72](https://github.com/appscode/voyager/issues/72)
- expose NodePort like functionality to Ingress [\#68](https://github.com/appscode/voyager/issues/68)
- Cross Namespace Service Support [\#40](https://github.com/appscode/voyager/issues/40)
- Support health checks [\#38](https://github.com/appscode/voyager/issues/38)
- Support full spectrum of HAProxy rules [\#21](https://github.com/appscode/voyager/issues/21)
- Add user provided annotations in LoadBalancer in Service/Pods [\#105](https://github.com/appscode/voyager/pull/105) ([sadlil](https://github.com/sadlil))
- Feature weighted backend [\#77](https://github.com/appscode/voyager/pull/77) ([sadlil](https://github.com/sadlil))
- Update svc instead of Deleting svc [\#87](https://github.com/appscode/voyager/pull/87) ([sadlil](https://github.com/sadlil))
- Feature: backend rules [\#80](https://github.com/appscode/voyager/pull/80) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Update service in NodePort & LoadBalancer mode [\#86](https://github.com/appscode/voyager/issues/86)
- Fix ALPN negotiation [\#32](https://github.com/appscode/voyager/issues/32)
- Use annotations for backend weight [\#83](https://github.com/appscode/voyager/pull/83) ([sadlil](https://github.com/sadlil))
- Fix Loadbalancer Port Open Issues [\#99](https://github.com/appscode/voyager/pull/99) ([sadlil](https://github.com/sadlil))
- Ensure pod delete [\#97](https://github.com/appscode/voyager/pull/97) ([sadlil](https://github.com/sadlil))
- Update svc instead of Deleting svc [\#87](https://github.com/appscode/voyager/pull/87) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Allow free form HTTP rewriting [\#76](https://github.com/appscode/voyager/issues/76)
- Test NodePort mode [\#98](https://github.com/appscode/voyager/issues/98)
- Ensure pods are deleted before deleting RC / Deployment [\#96](https://github.com/appscode/voyager/issues/96)
- Test that previously open NodePort is not reassigned [\#95](https://github.com/appscode/voyager/issues/95)
- Use HAProxy 1.7.5 [\#90](https://github.com/appscode/voyager/issues/90)
- Document 1.5.5 milestone features [\#78](https://github.com/appscode/voyager/issues/78)
- Specify different services in a backend with weights [\#75](https://github.com/appscode/voyager/issues/75)

**Merged pull requests:**

- Update top readme file [\#112](https://github.com/appscode/voyager/pull/112) ([tamalsaha](https://github.com/tamalsaha))
- Update docs [\#111](https://github.com/appscode/voyager/pull/111) ([tamalsaha](https://github.com/tamalsaha))
- NodePort Tests, Annotations Documentation [\#110](https://github.com/appscode/voyager/pull/110) ([sadlil](https://github.com/sadlil))
- Change HAProxy image to 1.7.5-1.5.5 [\#93](https://github.com/appscode/voyager/pull/93) ([tamalsaha](https://github.com/tamalsaha))
- Rename Daemon type to HostPort [\#84](https://github.com/appscode/voyager/pull/84) ([tamalsaha](https://github.com/tamalsaha))
- Use appscode/errors v2 [\#81](https://github.com/appscode/voyager/pull/81) ([tamalsaha](https://github.com/tamalsaha))
- Avoid upgrade in operator docker image [\#109](https://github.com/appscode/voyager/pull/109) ([tamalsaha](https://github.com/tamalsaha))
- Use alpine as the base image for operator [\#107](https://github.com/appscode/voyager/pull/107) ([tamalsaha](https://github.com/tamalsaha))
- Add `go` and `glide` commands to developer docs [\#101](https://github.com/appscode/voyager/pull/101) ([julianvmodesto](https://github.com/julianvmodesto))
- Ensure forward secrecy [\#94](https://github.com/appscode/voyager/pull/94) ([tamalsaha](https://github.com/tamalsaha))
- Update docs to build HAProxy 1.7.5 [\#92](https://github.com/appscode/voyager/pull/92) ([tamalsaha](https://github.com/tamalsaha))
- Use HAProxy 1.7.5 [\#91](https://github.com/appscode/voyager/pull/91) ([tamalsaha](https://github.com/tamalsaha))
- Introduce NodePort mode [\#85](https://github.com/appscode/voyager/pull/85) ([tamalsaha](https://github.com/tamalsaha))
- Update 1.5.5 Documentation [\#82](https://github.com/appscode/voyager/pull/82) ([sadlil](https://github.com/sadlil))

## [1.5.4](https://github.com/appscode/voyager/tree/1.5.4) (2017-05-08)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.3...1.5.4)

**Fixed bugs:**

- Voyager pod is restarting itself when attached backend pod restarts [\#69](https://github.com/appscode/voyager/issues/69)
- Do not restart lb pod when backend pod restarts [\#70](https://github.com/appscode/voyager/pull/70) ([sadlil](https://github.com/sadlil))

**Merged pull requests:**

- Rename operator deployment to voyager-operator [\#71](https://github.com/appscode/voyager/pull/71) ([tamalsaha](https://github.com/tamalsaha))

## [1.5.3](https://github.com/appscode/voyager/tree/1.5.3) (2017-05-03)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.2...1.5.3)

**Implemented enhancements:**

- Support StatefulSet pod names in Voyager [\#14](https://github.com/appscode/voyager/issues/14)
- Ingress Hostname based traffic forwarding [\#66](https://github.com/appscode/voyager/pull/66) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- cloud-provider & cloud-name can't be always required [\#64](https://github.com/appscode/voyager/issues/64)

**Merged pull requests:**

- Prepare docs for 1.5.3 release [\#67](https://github.com/appscode/voyager/pull/67) ([tamalsaha](https://github.com/tamalsaha))
- cloud-provider & cloud-name is not required for unknown providers. [\#65](https://github.com/appscode/voyager/pull/65) ([tamalsaha](https://github.com/tamalsaha))
- Test/fix ingress name [\#63](https://github.com/appscode/voyager/pull/63) ([ashiquzzaman33](https://github.com/ashiquzzaman33))
- Update docs to new chart location [\#60](https://github.com/appscode/voyager/pull/60) ([tamalsaha](https://github.com/tamalsaha))
- Move chart to root directory [\#59](https://github.com/appscode/voyager/pull/59) ([tamalsaha](https://github.com/tamalsaha))

## [1.5.2](https://github.com/appscode/voyager/tree/1.5.2) (2017-04-21)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.1...1.5.2)

**Implemented enhancements:**

- Add Retry on DaemonMode Loadbalancer http test call [\#52](https://github.com/appscode/voyager/pull/52) ([sadlil](https://github.com/sadlil))
- Fix Documentation [\#51](https://github.com/appscode/voyager/pull/51) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Slack channel token\_revoked [\#48](https://github.com/appscode/voyager/issues/48)
- Service ports should be int [\#47](https://github.com/appscode/voyager/issues/47)

**Merged pull requests:**

- Add service to deployments.yaml [\#58](https://github.com/appscode/voyager/pull/58) ([tamalsaha](https://github.com/tamalsaha))
- Prepare docs for version 1.5.2 [\#57](https://github.com/appscode/voyager/pull/57) ([tamalsaha](https://github.com/tamalsaha))
- Add service in voyager [\#56](https://github.com/appscode/voyager/pull/56) ([saumanbiswas](https://github.com/saumanbiswas))
- Fix stable chart [\#55](https://github.com/appscode/voyager/pull/55) ([saumanbiswas](https://github.com/saumanbiswas))
- Use unversioned time. [\#54](https://github.com/appscode/voyager/pull/54) ([tamalsaha](https://github.com/tamalsaha))
- Doc/fix update [\#53](https://github.com/appscode/voyager/pull/53) ([sadlil](https://github.com/sadlil))
- Initial voyager chart [\#43](https://github.com/appscode/voyager/pull/43) ([saumanbiswas](https://github.com/saumanbiswas))

## [1.5.1](https://github.com/appscode/voyager/tree/1.5.1) (2017-04-05)
[Full Changelog](https://github.com/appscode/voyager/compare/1.5.0...1.5.1)

**Implemented enhancements:**

- Enable GKE [\#44](https://github.com/appscode/voyager/issues/44)

**Merged pull requests:**

- Enable GKE [\#45](https://github.com/appscode/voyager/pull/45) ([tamalsaha](https://github.com/tamalsaha))
- Fix Typos [\#42](https://github.com/appscode/voyager/pull/42) ([sunkuet02](https://github.com/sunkuet02))
- update README [\#41](https://github.com/appscode/voyager/pull/41) ([ben-st](https://github.com/ben-st))

## [1.5.0](https://github.com/appscode/voyager/tree/1.5.0) (2017-03-01)
**Implemented enhancements:**

- Various clean ups [\#18](https://github.com/appscode/voyager/issues/18)
- Add ALPN options to TCP Backends [\#35](https://github.com/appscode/voyager/pull/35) ([sadlil](https://github.com/sadlil))
- Update docs with voyager options and test modes [\#34](https://github.com/appscode/voyager/pull/34) ([sadlil](https://github.com/sadlil))
- Add alpn option while TLS is used [\#25](https://github.com/appscode/voyager/pull/25) ([sadlil](https://github.com/sadlil))
- Adding Tests - Unit and E2E [\#12](https://github.com/appscode/voyager/pull/12) ([sadlil](https://github.com/sadlil))
- Ensure TPR at runtime [\#9](https://github.com/appscode/voyager/pull/9) ([sadlil](https://github.com/sadlil))
- add ingress-class [\#4](https://github.com/appscode/voyager/pull/4) ([sadlil](https://github.com/sadlil))
- Renamed ingress annotations to "ingress.appscode.com" [\#3](https://github.com/appscode/voyager/pull/3) ([sadlil](https://github.com/sadlil))
- use updated reloader. [\#2](https://github.com/appscode/voyager/pull/2) ([sadlil](https://github.com/sadlil))

**Fixed bugs:**

- Failing to deploy [\#29](https://github.com/appscode/voyager/issues/29)
- Remove ALPN h2 for https [\#33](https://github.com/appscode/voyager/pull/33) ([sadlil](https://github.com/sadlil))
- Update doc fix for \#19 [\#26](https://github.com/appscode/voyager/pull/26) ([sadlil](https://github.com/sadlil))

**Closed issues:**

- Update documentation for nodeSelector cleanup [\#24](https://github.com/appscode/voyager/issues/24)

**Merged pull requests:**

- Add doc explaining release process. [\#37](https://github.com/appscode/voyager/pull/37) ([tamalsaha](https://github.com/tamalsaha))
- Pass KLOADER\_ARGS as env variable [\#30](https://github.com/appscode/voyager/pull/30) ([tamalsaha](https://github.com/tamalsaha))
- Init cloud provider for Azure. [\#28](https://github.com/appscode/voyager/pull/28) ([tamalsaha](https://github.com/tamalsaha))
- Revendor dependencies. [\#23](https://github.com/appscode/voyager/pull/23) ([tamalsaha](https://github.com/tamalsaha))
- Use Ubuntu:16.04 as the base image to enable ALPN. [\#22](https://github.com/appscode/voyager/pull/22) ([tamalsaha](https://github.com/tamalsaha))
- Resolve \#18 [\#19](https://github.com/appscode/voyager/pull/19) ([sadlil](https://github.com/sadlil))
- Add example on front page. [\#16](https://github.com/appscode/voyager/pull/16) ([tamalsaha](https://github.com/tamalsaha))
- README typos [\#15](https://github.com/appscode/voyager/pull/15) ([JakeAustwick](https://github.com/JakeAustwick))
- Add links of subsections [\#11](https://github.com/appscode/voyager/pull/11) ([sadlil](https://github.com/sadlil))
- Update docs [\#10](https://github.com/appscode/voyager/pull/10) ([sadlil](https://github.com/sadlil))
- Rename voyager to Voyager  [\#8](https://github.com/appscode/voyager/pull/8) ([sadlil](https://github.com/sadlil))
- Add acknowledgements [\#7](https://github.com/appscode/voyager/pull/7) ([sadlil](https://github.com/sadlil))
- Documentation for voyager [\#6](https://github.com/appscode/voyager/pull/6) ([sadlil](https://github.com/sadlil))
- Use kloader. [\#5](https://github.com/appscode/voyager/pull/5) ([tamalsaha](https://github.com/tamalsaha))
- Custom pongo2 filters for loading haproxy data. [\#1](https://github.com/appscode/voyager/pull/1) ([sadlil](https://github.com/sadlil))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*