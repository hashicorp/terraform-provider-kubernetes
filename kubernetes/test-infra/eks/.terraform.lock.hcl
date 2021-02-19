# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "localhost/test/helm" {
  version     = "9.9.9"
  constraints = "9.9.9"
  hashes = [
    "h1:npZpS0MeIa75Fqq2LaEKcYZ8oi6rOogciOYH+oaCtvY=",
  ]
}

provider "localhost/test/kubernetes" {
  version     = "9.9.9"
  constraints = "9.9.9"
  hashes = [
    "h1:SWuBXYSfT0iDYpGWgBf/EhnzjCqlsuMQyFRl/co1Uyg=",
  ]
}

provider "registry.terraform.io/hashicorp/aws" {
  version     = "3.22.0"
  constraints = ">= 3.22.0, 3.22.0"
  hashes = [
    "h1:8aWXjFcmEi64P0TMHOCQXWws+/SmvJQrNvHlzdktKOM=",
    "zh:4a9a66caf1964cdd3b61fb3ebb0da417195a5529cb8e496f266b0778335d11c8",
    "zh:514f2f006ae68db715d86781673faf9483292deab235c7402ff306e0e92ea11a",
    "zh:5277b61109fddb9011728f6650ef01a639a0590aeffe34ed7de7ba10d0c31803",
    "zh:67784dc8c8375ab37103eea1258c3334ee92be6de033c2b37e3a2a65d0005142",
    "zh:76d4c8be2ca4a3294fb51fb58de1fe03361d3bc403820270cc8e71a04c5fa806",
    "zh:8f90b1cfdcf6e8fb1a9d0382ecaa5056a3a84c94e313fbf9e92c89de271cdede",
    "zh:d0ac346519d0df124df89be2d803eb53f373434890f6ee3fb37112802f9eac59",
    "zh:d6256feedada82cbfb3b1dd6dd9ad02048f23120ab50e6146a541cb11a108cc1",
    "zh:db2fe0d2e77c02e9a74e1ed694aa352295a50283f9a1cf896e5be252af14e9f4",
    "zh:eda61e889b579bd90046939a5b40cf5dc9031fb5a819fc3e4667a78bd432bdb2",
  ]
}

provider "registry.terraform.io/hashicorp/helm" {
  version = "2.0.3"
  hashes = [
    "h1:eUr4dHyxlcLmLja0wBgJC7t5bfHzbtACyuumKPuDrGs=",
    "zh:154e0aa489e474e2eeb3de94be7666133faf6fd950712a640425b2bf3a81ee95",
    "zh:16a2be6c4b61d0c5205c63816148c7ab0c8f56a75c05e8d897fa4d5cac0c029a",
    "zh:189e47bc723f8c29bcfe2c1638d43b8148f614ea86e642f4b50b2acb4b760224",
    "zh:3763901d3630213002cb8c70bb24c628cd29738ff6591585250ea8636264abd6",
    "zh:4822f85e4700ea049384523d98de0ef7d83549844b13e94bbd544cec05557a9a",
    "zh:62c5b87b09e0051bab0b712e3ad465fd53e66f9619dbe76ee23519d1087d8a05",
    "zh:a0a6a842b11190dd1841e98bbb74961074e7ffb95984be5cc392df9f532d803e",
    "zh:beac4e6806e77447e1018f3404a5fbf782d20d82a0d9b4a31e9bfc7d2bbecab6",
    "zh:e1bbaa09bf4f4a91ec7606f84d2e0200a02e7b24d045e8b5daebd87d7a75b7ce",
    "zh:ed1e05c50212d4f57435ccdd68cfb98d8395927c316df76d1dd6509566d3aeaa",
    "zh:fdc687e16a964bb652ddb670f6832fdead25235eca551796cfed70ec07d94931",
  ]
}

provider "registry.terraform.io/hashicorp/kubernetes" {
  version     = "2.0.2"
  constraints = ">= 1.11.1"
  hashes = [
    "h1:PRfDnUFBD4ud7SgsMAa5S2Gd60FeriD1PWE6EifjXB0=",
    "zh:4e66d509c828b0a2e599a567ad470bf85ebada62788aead87a8fb621301dec55",
    "zh:55ca6466a82f60d2c9798d171edafacc9ea4991aa7aa32ed5d82d6831cf44542",
    "zh:65741e6910c8b1322d9aef5dda4d98d1e6409aebc5514b518f46019cd06e1b47",
    "zh:79456ca037c19983977285703f19f4b04f7eadcf8eb6af21f5ea615026271578",
    "zh:7c39ced4dc44181296721715005e390021770077012c206ab4c209fb704b34d0",
    "zh:86856c82a6444c19b3e3005e91408ac68eb010c9218c4c4119fc59300b107026",
    "zh:999865090c72fa9b85c45e76b20839da51714ae429d1ab14b7d8ce66c2655abf",
    "zh:a3ea0ae37c61b4bfe81f7a395fb7b5ba61564e7d716d7a191372c3c983271d13",
    "zh:d9061861822933ebb2765fa691aeed2930ee495bfb6f72a5bdd88f43ccd9e038",
    "zh:e04adbe0d5597d1fdd4f418be19c9df171f1d709009f63b8ce1239b71b4fa45a",
  ]
}

provider "registry.terraform.io/hashicorp/local" {
  version     = "2.1.0"
  constraints = ">= 1.4.0"
  hashes = [
    "h1:EYZdckuGU3n6APs97nS2LxZm3dDtGqyM4qaIvsmac8o=",
    "zh:0f1ec65101fa35050978d483d6e8916664b7556800348456ff3d09454ac1eae2",
    "zh:36e42ac19f5d68467aacf07e6adcf83c7486f2e5b5f4339e9671f68525fc87ab",
    "zh:6db9db2a1819e77b1642ec3b5e95042b202aee8151a0256d289f2e141bf3ceb3",
    "zh:719dfd97bb9ddce99f7d741260b8ece2682b363735c764cac83303f02386075a",
    "zh:7598bb86e0378fd97eaa04638c1a4c75f960f62f69d3662e6d80ffa5a89847fe",
    "zh:ad0a188b52517fec9eca393f1e2c9daea362b33ae2eb38a857b6b09949a727c1",
    "zh:c46846c8df66a13fee6eff7dc5d528a7f868ae0dcf92d79deaac73cc297ed20c",
    "zh:dc1a20a2eec12095d04bf6da5321f535351a594a636912361db20eb2a707ccc4",
    "zh:e57ab4771a9d999401f6badd8b018558357d3cbdf3d33cc0c4f83e818ca8e94b",
    "zh:ebdcde208072b4b0f8d305ebf2bfdc62c926e0717599dcf8ec2fd8c5845031c3",
    "zh:ef34c52b68933bedd0868a13ccfd59ff1c820f299760b3c02e008dc95e2ece91",
  ]
}

provider "registry.terraform.io/hashicorp/null" {
  version     = "3.1.0"
  constraints = ">= 2.1.0"
  hashes = [
    "h1:vpC6bgUQoJ0znqIKVFevOdq+YQw42bRq0u+H3nto8nA=",
    "zh:02a1675fd8de126a00460942aaae242e65ca3380b5bb192e8773ef3da9073fd2",
    "zh:53e30545ff8926a8e30ad30648991ca8b93b6fa496272cd23b26763c8ee84515",
    "zh:5f9200bf708913621d0f6514179d89700e9aa3097c77dac730e8ba6e5901d521",
    "zh:9ebf4d9704faba06b3ec7242c773c0fbfe12d62db7d00356d4f55385fc69bfb2",
    "zh:a6576c81adc70326e4e1c999c04ad9ca37113a6e925aefab4765e5a5198efa7e",
    "zh:a8a42d13346347aff6c63a37cda9b2c6aa5cc384a55b2fe6d6adfa390e609c53",
    "zh:c797744d08a5307d50210e0454f91ca4d1c7621c68740441cf4579390452321d",
    "zh:cecb6a304046df34c11229f20a80b24b1603960b794d68361a67c5efe58e62b8",
    "zh:e1371aa1e502000d9974cfaff5be4cfa02f47b17400005a16f14d2ef30dc2a70",
    "zh:fc39cc1fe71234a0b0369d5c5c7f876c71b956d23d7d6f518289737a001ba69b",
    "zh:fea4227271ebf7d9e2b61b89ce2328c7262acd9fd190e1fd6d15a591abfa848e",
  ]
}

provider "registry.terraform.io/hashicorp/random" {
  version     = "3.1.0"
  constraints = ">= 2.1.0"
  hashes = [
    "h1:BZMEPucF+pbu9gsPk0G0BHx7YP04+tKdq2MrRDF1EDM=",
    "zh:2bbb3339f0643b5daa07480ef4397bd23a79963cc364cdfbb4e86354cb7725bc",
    "zh:3cd456047805bf639fbf2c761b1848880ea703a054f76db51852008b11008626",
    "zh:4f251b0eda5bb5e3dc26ea4400dba200018213654b69b4a5f96abee815b4f5ff",
    "zh:7011332745ea061e517fe1319bd6c75054a314155cb2c1199a5b01fe1889a7e2",
    "zh:738ed82858317ccc246691c8b85995bc125ac3b4143043219bd0437adc56c992",
    "zh:7dbe52fac7bb21227acd7529b487511c91f4107db9cc4414f50d04ffc3cab427",
    "zh:a3a9251fb15f93e4cfc1789800fc2d7414bbc18944ad4c5c98f466e6477c42bc",
    "zh:a543ec1a3a8c20635cf374110bd2f87c07374cf2c50617eee2c669b3ceeeaa9f",
    "zh:d9ab41d556a48bd7059f0810cf020500635bfc696c9fc3adab5ea8915c1d886b",
    "zh:d9e13427a7d011dbd654e591b0337e6074eef8c3b9bb11b2e39eaaf257044fd7",
    "zh:f7605bd1437752114baf601bdf6931debe6dc6bfe3006eb7e9bb9080931dca8a",
  ]
}

provider "registry.terraform.io/hashicorp/template" {
  version     = "2.2.0"
  constraints = ">= 2.1.0"
  hashes = [
    "h1:94qn780bi1qjrbC3uQtjJh3Wkfwd5+tTtJHOb7KTg9w=",
    "zh:01702196f0a0492ec07917db7aaa595843d8f171dc195f4c988d2ffca2a06386",
    "zh:09aae3da826ba3d7df69efeb25d146a1de0d03e951d35019a0f80e4f58c89b53",
    "zh:09ba83c0625b6fe0a954da6fbd0c355ac0b7f07f86c91a2a97849140fea49603",
    "zh:0e3a6c8e16f17f19010accd0844187d524580d9fdb0731f675ffcf4afba03d16",
    "zh:45f2c594b6f2f34ea663704cc72048b212fe7d16fb4cfd959365fa997228a776",
    "zh:77ea3e5a0446784d77114b5e851c970a3dde1e08fa6de38210b8385d7605d451",
    "zh:8a154388f3708e3df5a69122a23bdfaf760a523788a5081976b3d5616f7d30ae",
    "zh:992843002f2db5a11e626b3fc23dc0c87ad3729b3b3cff08e32ffb3df97edbde",
    "zh:ad906f4cebd3ec5e43d5cd6dc8f4c5c9cc3b33d2243c89c5fc18f97f7277b51d",
    "zh:c979425ddb256511137ecd093e23283234da0154b7fa8b21c2687182d9aea8b2",
  ]
}
