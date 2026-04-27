import { ru as ruTranslations } from "./ru";

export { ruTranslations as ru };

export type Translation = { [K in keyof typeof ruTranslations]: string };
