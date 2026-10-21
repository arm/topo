.metadata.component["bom-ref"] as $topo
| if ($topo == null or $topo == "") then error("Missing Topo bom-ref") else . end
| .components += [{
    type: "application",
    "bom-ref": "remoteproc-runtime",
    name: "remoteproc-runtime",
    scope: "optional",
    description: "Installed separately with containerd-shim-remoteproc-v1 by topo install remoteproc-runtime. The latest version is resolved at installation time; not bundled with Topo.",
    externalReferences: [{type: "distribution", url: "https://artifacts.tools.arm.com/remoteproc-runtime/"}]
  }]
| .dependencies = (.dependencies // [])
| if any(.dependencies[]; .ref == $topo) then
    (.dependencies[] | select(.ref == $topo) | .dependsOn) += ["remoteproc-runtime"]
  else
    .dependencies += [{ref: $topo, dependsOn: ["remoteproc-runtime"]}]
  end
