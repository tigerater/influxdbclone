local Label(name, desc, color) = {
    kind: 'Label',
    name: name,
    description: desc,
    color: color
};

local LabelAssociations(names=[]) = [
    {kind: 'Label', name: name}
    for name in names
];

local Bucket(name, desc, secs, associations=LabelAssociations(['label_1'])) = {
    kind: 'Bucket',
    name: name,
    description: desc,
    retentionRules: [
        {type: 'expire', everySeconds:  secs}
    ],
    associations: associations
};

{
   apiVersion: "0.1.0",
   kind: "Package",
   meta: {
     pkgName: "pkg_name",
     pkgVersion: "1",
     description: "pack description"
   },
   spec: {
     resources: [
        Label("label_1",desc="desc_1", color='#eee888'),
        Bucket(name="rucket_1", desc="desc_1", secs=10000),
        Bucket("rucket_2", "desc_2", 20000),
        Bucket("rucket_3", "desc_3", 30000),
     ]
   }
}
