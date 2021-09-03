import { atom } from 'recoil';
import { recoilPersist } from 'recoil-persist';

const { persistAtom } = recoilPersist()

const idTokenState = atom({
    key: 'idTokenState',
    default: '',
    effects_UNSTABLE: [persistAtom],
});

export { 
    idTokenState,
}