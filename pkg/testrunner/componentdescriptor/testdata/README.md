# Testdata

The ocm-repo-ctf, ocm-repo-ctf-partial-1 and ocm-repo-ctf-partial-2 ([here](./repositories)) are ocm ctf's (common 
transport archives) containing all the components present in [components](./components). The ocm library implements the 
same interface for ctf as it does for oci registries which makes it a convenient repository representation for tests.

#### All components in a single repository
Thereby, the root-component is the component described in the root-component.yaml and the other components are contained
in the ocm-repo-ctf.

The components span the following graph:

![component-graph](graph.png)

#### Components distributed over multiple repositories
In this case, the root-component is also the component described in the root-component.yaml.The union of the 
components in the partial repositories is equivalent to the components in the ocm-repo-ctf, but to resolve the entire 
dependency graph, multiple repositories have to be accessed.

**ocm-repo-ctf-partial-1:** {component-1, component-1-1, component-2-2}  
**ocm-repo-ctf-partial-2:** {component-2, component-2-1, component-3}

Generally, in both cases, the same set of components should be returned.

#### Updating components

After changing a component descriptor, it needs to be updated in the ocm-repo-ctf / partial-1 / partial-2 locations. 
Assuming the change was done to `component-1-1`, this can be done by running:

```bash
# Update the component descriptor in the ocm-repo-ctf
ocm transfer componentarchive ./components/component-1-1 ./repositories/ocm-repo-ctf --enforce
# Update the component descriptor in the ocm-repo-ctf-partial-1
ocm transfer componentarchive ./components/component-1-1 ./repositories/ocm-repo-ctf-partial-1 --enforce
```
